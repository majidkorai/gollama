package manager

import (
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/majidkorai/gollama/pkg/llama"
	"github.com/majidkorai/gollama/pkg/model"
)

type Instance struct {
	Port          int       `json:"port"`
	Model         string    `json:"model"`
	BlobPath      string    `json:"blob_path"`
	PID           int       `json:"pid"`
	Status        string    `json:"status"`
	Ready         bool      `json:"ready"`
	TokensPerSec  float64   `json:"tokens_per_sec,omitempty"`
	Flags         []string  `json:"flags,omitempty"`
	StartedAt     time.Time `json:"started_at"`
	TotalTokens   int64     `json:"total_tokens"`
	LastActivity  time.Time `json:"last_activity"`
	GpuUtil       float64   `json:"gpu_util"`                   // max across GPUs (P2-T7)
	GpuUtilPerGPU []float64 `json:"gpu_util_per_gpu,omitempty"` // per-GPU breakdown (P2-T7)
	CpuPercent    float64   `json:"cpu_percent"`
	MemoryMB      float64   `json:"memory_mb"`
	DeviceSplit   string    `json:"device_split,omitempty"`
	Profile       string    `json:"profile,omitempty"`
	Type          string    `json:"type,omitempty"` // "text" or "image"

	// Liveness bookkeeping (P2-T3). Not serialized.
	Cmd      *exec.Cmd     `json:"-"` // process gollama launched (nil for recovered orphans)
	WaitDone chan struct{} `json:"-"` // closed when the process has fully exited (nil for orphans)
}

type Manager struct {
	mu        sync.Mutex
	instances map[int]*Instance
	nextPort  int
}

// NewManager returns a Manager and recovers orphaned llama-server processes
// left behind by a previous gollama run.
func NewManager() *Manager {
	m := newManager()
	m.recoverOrphans()
	return m
}

// NewManagerNoRecovery is NewManager without the orphan ps scan. Tests use
// it for hermetic managers: a dummy llama-server spawned by one test binary
// would otherwise be adopted (and later stopped) by a gollama manager in
// another test binary running on the same host.
func NewManagerNoRecovery() *Manager {
	return newManager()
}

func newManager() *Manager {
	model.LoadConfig()
	m := &Manager{
		instances: make(map[int]*Instance),
		nextPort:  8081,
	}

	// Auto-stop idle instances
	go func() {
		for {
			time.Sleep(10 * time.Second)
			cfg := model.LoadConfig()
			if cfg.IdleTTL <= 0 {
				continue
			}
			m.StopIdle(time.Duration(cfg.IdleTTL) * time.Minute)
		}
	}()

	// Periodically refresh process metrics
	go func() {
		for {
			time.Sleep(15 * time.Second)
			m.mu.Lock()
			var pids []int
			for _, inst := range m.instances {
				if inst.Status == "running" && inst.PID > 0 {
					pids = append(pids, inst.PID)
				}
			}
			m.mu.Unlock()
			if len(pids) == 0 {
				continue
			}
			// P2-T7: first CPU sample (Linux /proc/<pid>/stat); the second
			// comes 1s later, giving an instantaneous % instead of ps's
			// lifetime average.
			var cpuFirst map[int]float64
			if runtime.GOOS == "linux" {
				cpuFirst = make(map[int]float64, len(pids))
				for _, pid := range pids {
					if ticks, ok := procCPUTicks(pid); ok {
						cpuFirst[pid] = ticks
					}
				}
				time.Sleep(time.Second)
			}
			m.mu.Lock()
			for _, inst := range m.instances {
				if inst.Status != "running" || inst.PID <= 0 {
					continue
				}
				applyCpuMetrics(inst, cpuFirst)
				if per, ok := queryGpuUtil(); ok {
					setGpuUtil(inst, per)
				}
			}
			m.mu.Unlock()
		}
	}()

	return m
}

func (m *Manager) recoverOrphans() {
	if runtime.GOOS == "windows" {
		m.recoverOrphansWindows()
		return
	}

	// Scan full command lines (ps -eo pid,args) instead of matching process
	// names (pgrep): for a shebang script the kernel records the interpreter
	// (e.g. /bin/sh) as the executable name, so pgrep never matched
	// "llama-server" on macOS and orphans were silently unrecovered, letting
	// a second 'gollama run' collide on the same port.
	out, err := exec.Command("ps", "-eo", "pid,args").Output()
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		pid, _ := strconv.Atoi(fields[0])
		if pid == 0 || pid == os.Getpid() {
			continue
		}
		cmdLine := strings.Join(fields[1:], " ")
		if !isLlamaServerCommandLine(cmdLine) {
			continue
		}
		m.registerOrphan(pid, cmdLine)
	}
}

// isLlamaServerCommandLine reports whether a full command line belongs to a
// llama-server process started by gollama: an argument whose basename is
// llama-server (or .exe) plus the --host flag gollama always passes.
func isLlamaServerCommandLine(cmdLine string) bool {
	if !strings.Contains(cmdLine, "--host ") {
		return false
	}
	for _, a := range strings.Fields(cmdLine) {
		base := filepath.Base(a)
		if base == "llama-server" || base == "llama-server.exe" {
			return true
		}
	}
	return false
}

// registerOrphan parses the port/model out of a recovered command line and
// registers the instance. Caller holds m.mu.
func (m *Manager) registerOrphan(pid int, cmdLine string) {
	var port int
	var modelName string
	args := strings.Fields(cmdLine)
	for i, a := range args {
		if a == "--port" && i+1 < len(args) {
			port, _ = strconv.Atoi(args[i+1])
		}
		if a == "-m" && i+1 < len(args) {
			modelName = args[i+1]
			if idx := strings.LastIndex(modelName, "/"); idx >= 0 {
				modelName = modelName[idx+1:]
			}
		}
	}
	if port == 0 {
		port = m.nextPort
		m.nextPort++
	}
	if port >= m.nextPort {
		m.nextPort = port + 1
	}
	if _, exists := m.instances[port]; !exists && port > 0 && !m.pidExists(pid) {
		m.instances[port] = &Instance{
			Port:         port,
			Model:        modelName,
			PID:          pid,
			Status:       "running",
			Ready:        true,
			LastActivity: time.Now(),
		}
		log.Printf("recovered orphan instance: port=%d pid=%d model=%s", port, pid, modelName)
	}
}

func (m *Manager) recoverOrphansWindows() {
	cmd := exec.Command("tasklist", "/FI", "IMAGENAME eq llama-server.exe", "/FO", "CSV", "/NH")
	out, err := cmd.Output()
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) < 2 {
			continue
		}
		pidStr := strings.Trim(parts[1], "\"")
		pid, _ := strconv.Atoi(pidStr)
		if pid == 0 {
			continue
		}

		m.recoverOrphanPidWindows(pid)
	}
}

// recoverOrphanPidWindows recovers one orphan by PID. tasklist CSV has no
// command line, so we ask WMI for it. P2-T6: when WMI fails (or the command
// line is not a gollama one) the process is skipped — the old code registered
// a phantom instance on a guessed port, which burned ports and showed fake
// instances in the UI.
func (m *Manager) recoverOrphanPidWindows(pid int) {
	if m.pidExists(pid) {
		return
	}
	wmi := exec.Command("wmic", "process", "where", fmt.Sprintf("ProcessId=%d", pid), "get", "CommandLine", "/format:value")
	wmiOut, wmiErr := wmi.Output()
	if wmiErr != nil {
		log.Printf("orphan recovery: cannot read command line of pid %d (%v) — skipping", pid, wmiErr)
		return
	}
	for _, wmiLine := range strings.Split(string(wmiOut), "\n") {
		wmiLine = strings.TrimSpace(wmiLine)
		if strings.HasPrefix(wmiLine, "CommandLine=") {
			m.registerOrphanFromCommandLine(pid, strings.TrimPrefix(wmiLine, "CommandLine="))
		}
	}
}

// registerOrphanFromCommandLine parses a process command line and registers
// an orphan instance when it looks gollama-launched (identified by the
// --host flag). Returns true when an instance was registered.
func (m *Manager) registerOrphanFromCommandLine(pid int, cmdLine string) bool {
	if m.pidExists(pid) {
		return false
	}
	if !strings.Contains(cmdLine, "--host ") {
		return false
	}
	var port int
	var modelName string
	args := strings.Fields(cmdLine)
	for i, a := range args {
		if a == "--port" && i+1 < len(args) {
			port, _ = strconv.Atoi(args[i+1])
		}
		if a == "-m" && i+1 < len(args) {
			modelName = args[i+1]
			if idx := strings.LastIndex(modelName, "/"); idx >= 0 {
				modelName = modelName[idx+1:]
			}
			if idx := strings.LastIndex(modelName, "\\"); idx >= 0 {
				modelName = modelName[idx+1:]
			}
		}
	}
	if port == 0 {
		port = m.nextPort
		m.nextPort++
	}
	if port >= m.nextPort {
		m.nextPort = port + 1
	}
	if _, exists := m.instances[port]; exists {
		return false
	}
	m.instances[port] = &Instance{
		Port:         port,
		Model:        modelName,
		PID:          pid,
		Status:       "running",
		LastActivity: time.Now(),
	}
	log.Printf("recovered orphan instance: port=%d pid=%d model=%s", port, pid, modelName)
	return true
}

func (m *Manager) RecoverOrphans() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.recoverOrphans()
}

func (m *Manager) pidExists(pid int) bool {
	for _, inst := range m.instances {
		if inst.PID == pid {
			return true
		}
	}
	return false
}

func portAvailable(port int) bool {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

func (m *Manager) Start(modelName string, port int, extraArgs []string, replaceFlags bool, profileEnv map[string]string, binaryPath ...string) (*Instance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Build set of flag keys from extraArgs early to check for --port
	extraKeys := make(map[string]bool)
	for _, a := range extraArgs {
		if strings.HasPrefix(a, "--") {
			extraKeys[a] = true
		}
	}

	// Resolve port: from extraArgs first, then parameter, then auto-assign
	if extraKeys["--port"] {
		for i, a := range extraArgs {
			if a == "--port" && i+1 < len(extraArgs) {
				if p, err := strconv.Atoi(extraArgs[i+1]); err == nil {
					port = p
				}
				break
			}
		}
	}
	if port == 0 {
		port = m.nextPort
		m.nextPort++
	}

	if existing, exists := m.instances[port]; exists {
		if existing.Status == "running" {
			return nil, fmt.Errorf("port %d is already in use", port)
		}
		// Stale entry (the process exited or was stopped) — reuse the slot
		// instead of permanently blocking this port.
		delete(m.instances, port)
	}

	if !portAvailable(port) {
		// Port might be in TIME_WAIT after a restart — retry for a bit
		for retry := 0; retry < 15; retry++ {
			time.Sleep(200 * time.Millisecond)
			if portAvailable(port) {
				break
			}
		}
	}
	if !portAvailable(port) {
		// Port taken by another process — find the next free one
		for i := port + 1; i < port+100; i++ {
			if _, exists := m.instances[i]; exists {
				continue
			}
			if portAvailable(i) {
				log.Printf("port %d is busy, using %d instead", port, i)
				port = i
				break
			}
		}
		if port > m.nextPort {
			m.nextPort = port + 1
		}
	}

	llamaBin := llama.FindLlamaServer()
	if len(binaryPath) > 0 && binaryPath[0] != "" {
		llamaBin = binaryPath[0]
	}
	blob, err := model.ResolveModelBlob(modelName)
	if err != nil {
		return nil, fmt.Errorf("resolving model: %w", err)
	}

	args := []string{"-m", blob}

	if replaceFlags {
		// UI sends the complete set — use only these flags
		args = append(args, extraArgs...)
		// Ensure --host and --port are always present
		hasHost, hasPort := false, false
		for _, a := range args {
			if a == "--host" { hasHost = true }
			if a == "--port" { hasPort = true }
		}
		if !hasHost { args = append(args, "--host", "0.0.0.0") }
		if !hasPort { args = append(args, "--port", strconv.Itoa(port)) }
	} else {
		// CLI sent partial flags — merge with defaults
		cfg := model.LoadConfig()
		for i := 0; i < len(cfg.DefaultFlags); i++ {
			a := cfg.DefaultFlags[i]
			if !strings.HasPrefix(a, "--") {
				// Orphaned value without a flag key — skip
				continue
			}
			isStandalone := model.IsStandaloneFlag(a)
			if extraKeys[a] {
				if i+1 < len(cfg.DefaultFlags) && !strings.HasPrefix(cfg.DefaultFlags[i+1], "--") && !isStandalone {
					i++
				}
				continue
			}
			args = append(args, a)
			if i+1 < len(cfg.DefaultFlags) && !strings.HasPrefix(cfg.DefaultFlags[i+1], "--") && !isStandalone {
				args = append(args, cfg.DefaultFlags[i+1])
				i++
			}
		}
		args = append(args, extraArgs...)

		// Ensure --host and --port are always present
		hasHost, hasPort := false, false
		for _, a := range args {
			if a == "--host" { hasHost = true }
			if a == "--port" { hasPort = true }
		}
		if !hasHost { args = append(args, "--host", "0.0.0.0") }
		if !hasPort { args = append(args, "--port", strconv.Itoa(port)) }

		// Auto-detect GPU if not specified (skip if user set --tensor-split for multi-GPU)
		hasGpuLayers := false
		hasTensorSplit := false
		for _, a := range args {
			if a == "--n-gpu-layers" || strings.HasPrefix(a, "--n-gpu-layers=") {
				hasGpuLayers = true
			}
			if a == "--tensor-split" {
				hasTensorSplit = true
			}
		}
		if !hasGpuLayers && !hasTensorSplit {
			if gpuAvailable, gpuLayers := model.DetectGPU(); gpuAvailable {
				args = append([]string{"--n-gpu-layers", strconv.Itoa(gpuLayers)}, args...)
				log.Printf("GPU detected, adding --n-gpu-layers %d", gpuLayers)
			}
		} else if hasTensorSplit {
			log.Printf("tensor-split detected, skipping auto --n-gpu-layers (let auto-fit decide)")
		}
	}

	log.Printf("launching llama-server: %s %s", llamaBin, strings.Join(args, " "))
	logDir := filepath.Join(model.GollamaDir(), "logs")
	model.EnsureDir(logDir)
	logFile := filepath.Join(logDir, fmt.Sprintf("port-%d.log", port))
	logF, logErr := os.Create(logFile)
	if logErr != nil {
		log.Printf("warning: could not create log file %s: %v", logFile, logErr)
		logF = nil
	}

	cmd := exec.Command(llamaBin, args...)
	binDir := model.BinDir()
	// If using a custom binary path, use its directory for library resolution
	if len(binaryPath) > 0 && binaryPath[0] != "" {
		binDir = filepath.Dir(llamaBin)
	}
	libVar := "LD_LIBRARY_PATH"
	if runtime.GOOS == "darwin" {
		libVar = "DYLD_LIBRARY_PATH"
	}
	libPath := binDir
	if existing := os.Getenv(libVar); existing != "" {
		libPath = binDir + ":" + existing
	}
	cmd.Env = append(os.Environ(), libVar+"="+libPath, "LC_ALL=C")
	for k, v := range profileEnv {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	if logF != nil {
		cmd.Stdout = io.MultiWriter(logF, os.Stderr)
		cmd.Stderr = io.MultiWriter(logF, os.Stderr)
	} else {
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting llama-server: %w", err)
	}

	inst := &Instance{
		Port:         port,
		Model:        modelName,
		BlobPath:     blob,
		PID:          cmd.Process.Pid,
		Status:       "running",
		Flags:        args,
		StartedAt:    time.Now(),
		LastActivity: time.Now(),
		Cmd:          cmd,
	}
	inst.WaitDone = make(chan struct{})

	// Calculate device split from launch flags and model metadata
	var ngl, ncpuMoe float64
	for i, a := range args {
		if (a == "--n-gpu-layers" || a == "--gpu-layers") && i+1 < len(args) {
			if args[i+1] == "all" || args[i+1] == "999" {
				ngl = 999
			} else if v, err := strconv.ParseFloat(args[i+1], 64); err == nil {
				ngl = v
			}
		}
		if a == "--n-cpu-moe" && i+1 < len(args) {
			if v, err := strconv.ParseFloat(args[i+1], 64); err == nil {
				ncpuMoe = v
			}
		}
	}
	totalBlocks := float64(model.ReadBlockCount(blob))
	if ngl > 0 {
		if totalBlocks == 0 {
			// No block count from GGUF — show raw flag values
			inst.DeviceSplit = fmt.Sprintf("GPU=%d layers", int(ngl))
			if ncpuMoe > 0 {
				inst.DeviceSplit += fmt.Sprintf(", CPU MoE=%d layers", int(ncpuMoe))
			}
		} else {
			gpuPct := math.Min(ngl/totalBlocks, 1.0) * 100
			cpuPct := 100.0 - gpuPct
			if ncpuMoe > 0 {
				moePct := math.Min(ncpuMoe/totalBlocks, 1.0) * 100
				inst.DeviceSplit = fmt.Sprintf("%.0f%% GPU / %.0f%% CPU (MoE: %.0f%%)", gpuPct-moePct, cpuPct+moePct, moePct)
			} else {
				inst.DeviceSplit = fmt.Sprintf("%.0f%% GPU / %.0f%% CPU", gpuPct, cpuPct)
			}
		}
	}
	m.instances[port] = inst

	log.Printf("instance started: model=%s port=%d pid=%d", modelName, port, cmd.Process.Pid)

	// Wait for instance to be ready before returning
	go func() {
		healthClient := &http.Client{Timeout: 2 * time.Second}
		baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
		deadline := time.Now().Add(model.LoadTimeout())
		for time.Now().Before(deadline) {
			resp, err := healthClient.Get(baseURL + "/health")
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == 200 {
					m.mu.Lock()
					inst.Ready = true
					// One-shot process metrics snapshot (stable after startup)
					cpu, mem := queryProcessStats(inst.PID)
					inst.CpuPercent = cpu
					inst.MemoryMB = mem
					if per, ok := queryGpuUtil(); ok {
						setGpuUtil(inst, per)
					}
					m.mu.Unlock()
					log.Printf("instance ready: port=%d cpu=%.0f%% mem=%.0fMB gpu=%.0f%%", port, cpu, mem, inst.GpuUtil)
					return
				}
			}
			time.Sleep(500 * time.Millisecond)
		}
		log.Printf("instance did not become ready: port=%d — check logs with 'gollama logs %d'", port, port)
	}()

	go func() {
		err := cmd.Wait()
		close(inst.WaitDone)
		m.mu.Lock()
		if inst.Status == "running" {
			inst.Status = "stopped"
			if err != nil {
				log.Printf("instance stopped with error: port=%d err=%v", port, err)
				// Read log tail to help diagnose
				if data, readErr := os.ReadFile(logFile); readErr == nil {
					lines := strings.Split(string(data), "\n")
					// Find the last non-empty lines (skip progress bars)
					for i := len(lines) - 1; i >= 0 && i > len(lines)-10; i-- {
						line := strings.TrimSpace(lines[i])
						if line != "" && !strings.Contains(line, "\r") {
							log.Printf("port=%d last log: %s", port, line)
							break
						}
					}
				}
			} else {
				log.Printf("instance stopped: port=%d", port)
			}
		}
		m.mu.Unlock()
	}()

	return inst, nil
}

// StartImage starts the Python image generation server as a managed subprocess.
func (m *Manager) StartImage(modelID string, port int, env map[string]string) (*Instance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if existing, exists := m.instances[port]; exists {
		if existing.Status == "running" {
			return nil, fmt.Errorf("port %d is already in use", port)
		}
		// Stale entry (the process exited or was stopped) — reuse the slot
		// instead of permanently blocking this port.
		delete(m.instances, port)
	}

	if !portAvailable(port) {
		for retry := 0; retry < 15; retry++ {
			time.Sleep(200 * time.Millisecond)
			if portAvailable(port) {
				break
			}
		}
	}
	if !portAvailable(port) {
		for i := port + 1; i < port+100; i++ {
			if _, exists := m.instances[i]; exists {
				continue
			}
			if portAvailable(i) {
				log.Printf("port %d is busy, using %d instead", port, i)
				port = i
				break
			}
		}
		if port > m.nextPort {
			m.nextPort = port + 1
		}
	}

	pythonBin := model.ImagePythonPath()
	appPath := model.ImageAppPath()

	if _, err := os.Stat(pythonBin); os.IsNotExist(err) {
		return nil, fmt.Errorf("Python binary not found at %s — set GOLLAMA_IMAGE_PYTHON env var", pythonBin)
	}
	if _, err := os.Stat(appPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("image app not found at %s — set GOLLAMA_IMAGE_APP env var", appPath)
	}

	cmd := exec.Command(pythonBin, appPath)
	cmd.Env = os.Environ()
	if modelID != "" {
		cmd.Env = append(cmd.Env, "MODEL_ID="+modelID)
	}
	cmd.Env = append(cmd.Env, "PORT="+strconv.Itoa(port))
	cmd.Env = append(cmd.Env, "HOST=127.0.0.1")
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	logDir := filepath.Join(model.GollamaDir(), "logs")
	model.EnsureDir(logDir)
	logFile := filepath.Join(logDir, fmt.Sprintf("port-%d.log", port))
	logF, logErr := os.Create(logFile)
	if logErr != nil {
		log.Printf("warning: could not create log file %s: %v", logFile, logErr)
		logF = nil
	}
	if logF != nil {
		cmd.Stdout = io.MultiWriter(logF, os.Stderr)
		cmd.Stderr = io.MultiWriter(logF, os.Stderr)
	} else {
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting image server: %w", err)
	}

	inst := &Instance{
		Port:         port,
		Model:        modelID,
		PID:          cmd.Process.Pid,
		Status:       "running",
		StartedAt:    time.Now(),
		LastActivity: time.Now(),
		Type:         "image",
		Cmd:          cmd,
	}
	inst.WaitDone = make(chan struct{})
	m.instances[port] = inst

	log.Printf("image instance started: model=%s port=%d pid=%d", modelID, port, cmd.Process.Pid)

	go func() {
		healthClient := &http.Client{Timeout: 2 * time.Second}
		baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
		deadline := time.Now().Add(model.LoadTimeout())
		for time.Now().Before(deadline) {
			resp, err := healthClient.Get(baseURL + "/health")
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == 200 {
					m.mu.Lock()
					inst.Ready = true
					m.mu.Unlock()
					log.Printf("image instance ready: port=%d", port)
					return
				}
			}
			time.Sleep(500 * time.Millisecond)
		}
		log.Printf("image instance did not become ready: port=%d — check logs with 'gollama logs %d'", port, port)
	}()

	go func() {
		err := cmd.Wait()
		close(inst.WaitDone)
		m.mu.Lock()
		if inst.Status == "running" {
			inst.Status = "stopped"
			if err != nil {
				log.Printf("image instance stopped with error: port=%d err=%v", port, err)
			} else {
				log.Printf("image instance stopped: port=%d", port)
			}
		}
		m.mu.Unlock()
	}()

	return inst, nil
}

// Stop signals the instance's process, waits up to 500ms for a graceful
// exit, and only then escalates to SIGKILL if the process is still alive.
// Liveness is observed through the cmd.Wait() goroutine (WaitDone) when
// gollama owns the process, or a signal-0 probe for recovered orphans —
// never a blind Kill, which could hit a recycled PID. The mutex is not
// held while waiting.
func (m *Manager) Stop(port int) error {
	m.mu.Lock()
	inst, ok := m.instances[port]
	m.mu.Unlock()
	if !ok {
		return fmt.Errorf("no instance on port %d", port)
	}

	var proc *os.Process
	if inst.Cmd != nil {
		proc = inst.Cmd.Process
	} else if inst.PID > 0 {
		proc, _ = os.FindProcess(inst.PID)
	}
	if proc != nil {
		proc.Signal(os.Interrupt) // ignore error: process may already be gone
	}

	// Give the process a moment to exit gracefully.
	if inst.WaitDone != nil {
		select {
		case <-inst.WaitDone:
		case <-time.After(500 * time.Millisecond):
		}
	} else {
		time.Sleep(500 * time.Millisecond)
	}

	// Still alive? Escalate to kill, then wait for the reap so the slot
	// is truly free.
	alive := false
	switch {
	case inst.WaitDone != nil:
		select {
		case <-inst.WaitDone:
		default:
			alive = true
		}
	case proc != nil && runtime.GOOS == "windows":
		alive = true // no signal-0 probe on Windows — escalate
	case proc != nil:
		alive = proc.Signal(syscall.Signal(0)) == nil
	}
	if alive {
		log.Printf("instance on port %d did not exit after SIGINT — sending SIGKILL", port)
		if proc != nil {
			proc.Kill()
		}
		if inst.WaitDone != nil {
			select {
			case <-inst.WaitDone:
			case <-time.After(2 * time.Second):
			}
		}
	}

	m.mu.Lock()
	inst.Status = "stopped"
	delete(m.instances, port)
	m.mu.Unlock()
	return nil
}

// StopAll stops every known instance (in parallel) and returns the ports
// that were stopped. Used on 'serve' shutdown so GPU memory is reclaimed
// when gollama exits.
func (m *Manager) StopAll() []int {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var stopped []int
	for _, inst := range m.List() {
		port := inst.Port
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := m.Stop(port); err == nil {
				mu.Lock()
				stopped = append(stopped, port)
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	return stopped
}

func (m *Manager) UpdateTokens(port int, tps float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if inst, ok := m.instances[port]; ok {
		inst.TokensPerSec = tps
	}
}

func (m *Manager) AddCompletionTokens(port int, n int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if inst, ok := m.instances[port]; ok {
		inst.TotalTokens += n
	}
}

func (m *Manager) SetProfile(port int, profile string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if inst, ok := m.instances[port]; ok {
		inst.Profile = profile
	}
}

func (m *Manager) HasInstance(port int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.instances[port]
	return ok
}

// InstanceStatus returns the current status of the instance on the given port,
// or "" if no such instance is known.
func (m *Manager) InstanceStatus(port int) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if inst, ok := m.instances[port]; ok {
		return inst.Status
	}
	return ""
}

func (m *Manager) TouchActivity(port int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.touchActivityLocked(port)
}

func (m *Manager) touchActivityLocked(port int) {
	if inst, ok := m.instances[port]; ok {
		inst.LastActivity = time.Now()
	}
}

func (m *Manager) StopIdle(ttl time.Duration) []int {
	m.mu.Lock()
	var ports []int
	for port, inst := range m.instances {
		if inst.Status != "running" {
			continue
		}
		// Check log file modification time for activity (catches direct requests)
		logFile := filepath.Join(model.GollamaDir(), "logs", fmt.Sprintf("port-%d.log", port))
		if fi, err := os.Stat(logFile); err == nil {
			if fi.ModTime().After(inst.LastActivity) {
				inst.LastActivity = fi.ModTime()
			}
		}
		if time.Since(inst.LastActivity) < ttl {
			continue
		}
		ports = append(ports, port)
	}
	m.mu.Unlock()
	for _, port := range ports {
		log.Printf("auto-stopping idle instance: port=%d (idle >= %v)", port, ttl)
		m.Stop(port)
	}
	return ports
}

// FindInstanceByModel returns the running instance for modelName, or nil.
// Matching is deterministic (P2-T5): candidates are scored
// exact/same-blob (0) > short-name (1) > suffix (2) > substring (3), and
// ties are broken by the lowest port — the old code returned whichever
// candidate map iteration hit first.
func (m *Manager) FindInstanceByModel(modelName string) *Instance {
	m.mu.Lock()
	defer m.mu.Unlock()

	if modelName == "" {
		return nil
	}

	// Deterministic iteration order: ascending port.
	ports := make([]int, 0, len(m.instances))
	for port := range m.instances {
		ports = append(ports, port)
	}
	sort.Ints(ports)

	idx := model.LoadIndex()
	queryBlob, _ := model.ResolveModelBlob(modelName)

	const (
		tierExact = iota // case-insensitive name match, or the same GGUF file
		tierShort        // short name of the served model matches
		tierSuffix
		tierSubstring
	)
	matchTier := func(inst *Instance) int {
		if strings.EqualFold(inst.Model, modelName) || (queryBlob != "" && inst.BlobPath == queryBlob) {
			return tierExact
		}
		if info, ok := idx[inst.Model]; ok && info.ShortName != "" && strings.EqualFold(info.ShortName, modelName) {
			return tierShort
		}
		if strings.HasSuffix(strings.ToLower(inst.Model), strings.ToLower(modelName)) {
			return tierSuffix
		}
		if containsIgnoreCase(inst.Model, modelName) {
			return tierSubstring
		}
		return -1
	}

	var best *Instance
	bestTier := tierSubstring + 1
	for _, port := range ports {
		inst := m.instances[port]
		if inst.Status != "running" {
			continue
		}
		if tier := matchTier(inst); tier >= 0 && tier < bestTier {
			best, bestTier = inst, tier
			if tier == tierExact {
				break
			}
		}
	}
	if best != nil && bestTier == tierSubstring {
		log.Printf("model %q matched running instance on port %d by substring only — consider an exact model or profile name", modelName, best.Port)
	}
	return best
}

func (m *Manager) instancesByModelLocked(modelName string) (*Instance, bool) {
	for _, inst := range m.instances {
		if inst.Status == "running" && strings.EqualFold(inst.Model, modelName) {
			return inst, true
		}
	}
	return nil, false
}

func containsIgnoreCase(s, substr string) bool {
	s, substr = strings.ToLower(s), strings.ToLower(substr)
	return strings.Contains(s, substr)
}

func (m *Manager) List() []*Instance {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]*Instance, 0, len(m.instances))
	for _, inst := range m.instances {
		result = append(result, inst)
	}
	return result
}

func queryProcessStats(pid int) (cpuPercent, memoryMB float64) {
	cmd := exec.Command("ps", "-o", "%cpu=,%mem=,rss=", "-p", strconv.Itoa(pid))
	out, err := cmd.Output()
	if err != nil {
		return 0, 0
	}
	line := strings.TrimSpace(string(out))
	if line == "" {
		return 0, 0
	}
	fields := strings.Fields(line)
	if len(fields) >= 1 {
		cpu, _ := strconv.ParseFloat(fields[0], 64)
		cpuPercent = cpu
	}
	if len(fields) >= 3 {
		if rssKB, err := strconv.ParseFloat(fields[2], 64); err == nil && rssKB > 0 {
			memoryMB = rssKB / 1024
		}
	}
	return
}

// queryGpuUtil returns per-GPU utilization percentages from nvidia-smi
// (one entry per GPU, Linux only). ok is false when no GPU data is
// available; the caller keeps the instance's last known values.
func queryGpuUtil() ([]float64, bool) {
	if runtime.GOOS != "linux" {
		return nil, false
	}
	cmd := exec.Command("nvidia-smi", "--query-gpu=utilization.gpu", "--format=csv,noheader,nounits")
	out, err := cmd.Output()
	if err != nil {
		return nil, false
	}
	return parseGpuUtilCSV(out)
}

// parseGpuUtilCSV parses nvidia-smi's utilization.gpu CSV output: one
// integer percentage per line, one line per GPU. Unparseable lines are
// skipped; no valid lines -> not ok.
func parseGpuUtilCSV(out []byte) ([]float64, bool) {
	var per []float64
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if u, err := strconv.ParseFloat(strings.TrimSpace(line), 64); err == nil {
			per = append(per, u)
		}
	}
	if len(per) == 0 {
		return nil, false
	}
	return per, true
}

// setGpuUtil applies a per-GPU utilization sample to the instance: GpuUtil
// becomes the max across devices and GpuUtilPerGPU the full breakdown
// (P2-T7). An empty sample is a no-op.
func setGpuUtil(inst *Instance, per []float64) {
	if len(per) == 0 {
		return
	}
	inst.GpuUtilPerGPU = per
	max := per[0]
	for _, u := range per[1:] {
		if u > max {
			max = u
		}
	}
	inst.GpuUtil = max
}

// applyCpuMetrics updates inst.CpuPercent and inst.MemoryMB. The CPU value
// is instantaneous when a 1-second /proc/<pid>/stat double-sample
// succeeded (Linux, P2-T7); ps's %cpu — a lifetime average — is the
// fallback and the memory source everywhere.
func applyCpuMetrics(inst *Instance, cpuFirst map[int]float64) {
	cpu, mem := queryProcessStats(inst.PID)
	if f, ok := cpuFirst[inst.PID]; ok {
		if s, ok2 := procCPUTicks(inst.PID); ok2 && s >= f {
			if tps := cpuTicksPerSec(); tps > 0 {
				cpu = (s - f) / tps
			}
		}
	}
	inst.CpuPercent = cpu
	inst.MemoryMB = mem
}

// parseProcStatTicks extracts utime+stime (in clock ticks) from a
// /proc/<pid>/stat line. comm (field 2) is wrapped in parentheses and may
// itself contain spaces and parentheses, so parsing starts after the last
// ')'. utime/stime are fields 14/15 overall, i.e. indexes 11/12 after the
// closing paren.
func parseProcStatTicks(data []byte) (float64, bool) {
	s := string(data)
	idx := strings.LastIndexByte(s, ')')
	if idx < 0 || idx+1 >= len(s) {
		return 0, false
	}
	fields := strings.Fields(s[idx+1:])
	if len(fields) < 13 {
		return 0, false
	}
	utime, err1 := strconv.ParseFloat(fields[11], 64)
	stime, err2 := strconv.ParseFloat(fields[12], 64)
	if err1 != nil || err2 != nil {
		return 0, false
	}
	return utime + stime, true
}
