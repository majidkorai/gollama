package manager

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/majidkorai/gollama/pkg/llama"
	"github.com/majidkorai/gollama/pkg/model"
)

type Instance struct {
	Port         int      `json:"port"`
	Model        string   `json:"model"`
	PID          int      `json:"pid"`
	Status       string   `json:"status"`
	TokensPerSec float64  `json:"tokens_per_sec,omitempty"`
	Flags        []string `json:"flags,omitempty"`
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
	return m
}

func (m *Manager) recoverOrphans() {
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
		if _, exists := m.instances[port]; !exists && port > 0 {
			m.instances[port] = &Instance{
				Port:   port,
				Model:  modelName,
				PID:    pid,
				Status: "running",
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

func portAvailable(port int) bool {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

func (m *Manager) Start(modelName string, port int, extraArgs []string) (*Instance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if port == 0 {
		port = m.nextPort
		m.nextPort++
	}

	if _, exists := m.instances[port]; exists {
		return nil, fmt.Errorf("port %d is already in use", port)
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

	args := []string{
		"-m", blob,
		"--host", "0.0.0.0",
		"--port", strconv.Itoa(port),
	}
	cfg := model.LoadConfig()
	args = append(args, cfg.DefaultFlags...)
	args = append(args, extraArgs...)

	log.Printf("launching llama-server: %s %s", llamaBin, strings.Join(args, " "))
	logDir := filepath.Join(model.GollamaDir(), "logs")
	model.EnsureDir(logDir)
	logFile := filepath.Join(logDir, fmt.Sprintf("port-%d.log", port))
	logF, logErr := os.Create(logFile)
	if logErr != nil {
		log.Printf("warning: could not create log file %s: %v", logFile, logErr)
	}

	cmd := exec.Command(llamaBin, args...)
	cmd.Stdout = logF
	cmd.Stderr = logF

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting llama-server: %w", err)
	}

	inst := &Instance{
		Port:   port,
		Model:  modelName,
		PID:    cmd.Process.Pid,
		Status: "running",
		Flags:  args,
	}
	m.instances[port] = inst

	log.Printf("instance started: model=%s port=%d pid=%d", modelName, port, cmd.Process.Pid)

	go func() {
		err := cmd.Wait()
		m.mu.Lock()
		if inst.Status == "running" {
			inst.Status = "stopped"
			if err != nil {
				log.Printf("instance stopped with error: port=%d err=%v", port, err)
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

func (m *Manager) List() []*Instance {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]*Instance, 0, len(m.instances))
	for _, inst := range m.instances {
		result = append(result, inst)
	}
	return result
}
