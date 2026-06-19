package manager

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/majidkorai/gollama/pkg/llama"
	"github.com/majidkorai/gollama/pkg/model"
)

type Instance struct {
	Port         int        `json:"port"`
	Model        string     `json:"model"`
	PID          int        `json:"pid"`
	Status       string     `json:"status"`
	Ready        bool       `json:"ready"`
	TokensPerSec float64    `json:"tokens_per_sec,omitempty"`
	Flags        []string   `json:"flags,omitempty"`
	StartedAt    time.Time  `json:"started_at"`
	TotalTokens  int64      `json:"total_tokens"`
	LastActivity time.Time  `json:"last_activity"`
}

type Manager struct {
	mu        sync.Mutex
	instances map[int]*Instance
	nextPort  int
}

func NewManager() *Manager {
	model.LoadConfig()
	m := &Manager{
		instances: make(map[int]*Instance),
		nextPort:  8081,
	}
	m.recoverOrphans()

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

	return m
}

func (m *Manager) recoverOrphans() {
	if runtime.GOOS == "windows" {
		m.recoverOrphansWindows()
		return
	}

	cmd := exec.Command("pgrep", "-a", "llama-server")
	out, err := cmd.Output()
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		pidStr := parts[0]
		pid, _ := strconv.Atoi(pidStr)
		if pid == 0 {
			continue
		}

		cmdLine := strings.Join(parts[1:], " ")
		// Only recover instances started by gollama (identified by --host flag)
		if !strings.Contains(cmdLine, "--host ") {
			continue
		}
		var port int
		var modelName string
		args := parts[1:]
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

		// On Windows we can't get command-line args from tasklist CSV,
		// so we try to match by looking at running processes' command lines via WMI
		wmi := exec.Command("wmic", "process", "where", fmt.Sprintf("ProcessId=%d", pid), "get", "CommandLine", "/format:value")
		wmiOut, wmiErr := wmi.Output()
		if wmiErr != nil {
			// Fallback: register with a guessed port, but only if PID not already tracked
			if m.pidExists(pid) {
				continue
			}
			port := m.nextPort
			m.nextPort++
		m.instances[port] = &Instance{
				Port:         port,
				Model:        "unknown",
				PID:          pid,
				Status:       "running",
				Ready:        true,
				LastActivity: time.Now(),
			}
			log.Printf("recovered orphan instance (limited): port=%d pid=%d", port, pid)
			continue
		}

		var port int
		var modelName string
		for _, wmiLine := range strings.Split(string(wmiOut), "\n") {
			wmiLine = strings.TrimSpace(wmiLine)
			if strings.HasPrefix(wmiLine, "CommandLine=") {
				cmdLine := strings.TrimPrefix(wmiLine, "CommandLine=")
				// Only recover instances started by gollama (identified by --host flag)
				if !strings.Contains(cmdLine, "--host ") {
					continue
				}
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
				LastActivity: time.Now(),
			}
			log.Printf("recovered orphan instance: port=%d pid=%d model=%s", port, pid, modelName)
		}
	}
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

func (m *Manager) Start(modelName string, port int, extraArgs []string, replaceFlags bool) (*Instance, error) {
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

	if _, exists := m.instances[port]; exists {
		return nil, fmt.Errorf("port %d is already in use", port)
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
			if extraKeys[a] {
				if i+1 < len(cfg.DefaultFlags) && !strings.HasPrefix(cfg.DefaultFlags[i+1], "--") {
					i++
				}
				continue
			}
			args = append(args, a)
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

		// Auto-detect GPU if not specified
		hasGpuLayers := false
		for _, a := range args {
			if a == "--n-gpu-layers" || strings.HasPrefix(a, "--n-gpu-layers=") {
				hasGpuLayers = true
				break
			}
		}
		if !hasGpuLayers {
			if gpuAvailable, gpuLayers := model.DetectGPU(); gpuAvailable {
				args = append([]string{"--n-gpu-layers", strconv.Itoa(gpuLayers)}, args...)
				log.Printf("GPU detected, adding --n-gpu-layers %d", gpuLayers)
			}
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
	libVar := "LD_LIBRARY_PATH"
	if runtime.GOOS == "darwin" {
		libVar = "DYLD_LIBRARY_PATH"
	}
	libPath := binDir
	if existing := os.Getenv(libVar); existing != "" {
		libPath = binDir + ":" + existing
	}
	cmd.Env = append(os.Environ(), libVar+"="+libPath)
	if logF != nil {
		cmd.Stdout = logF
		cmd.Stderr = logF
	} else {
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting llama-server: %w", err)
	}

	inst := &Instance{
		Port:         port,
		Model:        modelName,
		PID:          cmd.Process.Pid,
		Status:       "running",
		Flags:        args,
		StartedAt:    time.Now(),
		LastActivity: time.Now(),
	}
	m.instances[port] = inst

	log.Printf("instance started: model=%s port=%d pid=%d", modelName, port, cmd.Process.Pid)

	// Wait for instance to be ready before returning
	go func() {
		healthClient := &http.Client{Timeout: 2 * time.Second}
		baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
		deadline := time.Now().Add(120 * time.Second)
		for time.Now().Before(deadline) {
			resp, err := healthClient.Get(baseURL + "/health")
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == 200 {
					m.mu.Lock()
					inst.Ready = true
					m.mu.Unlock()
					log.Printf("instance ready: port=%d", port)
					return
				}
			}
			time.Sleep(500 * time.Millisecond)
		}
		log.Printf("instance did not become ready: port=%d — check logs with 'gollama logs %d'", port, port)
	}()

	go func() {
		err := cmd.Wait()
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

func (m *Manager) Stop(port int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	inst, ok := m.instances[port]
	if !ok {
		return fmt.Errorf("no instance on port %d", port)
	}

	proc, err := os.FindProcess(inst.PID)
	if err == nil {
		proc.Signal(os.Interrupt)
		proc.Kill()
	}

	inst.Status = "stopped"
	delete(m.instances, port)
	return nil
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

func (m *Manager) FindInstanceByModel(modelName string) *Instance {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Exact match first
	if inst, ok := m.instancesByModelLocked(modelName); ok {
		return inst
	}
	// Suffix match (e.g. "qwen2.5-7b" matches "hf.co/Qwen/Qwen2.5-7B-Instruct-GGUF:Q4_K_M")
	for _, inst := range m.instances {
		if inst.Status == "running" && containsIgnoreCase(inst.Model, modelName) {
			return inst
		}
	}
	return nil
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
