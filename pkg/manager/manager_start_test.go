//go:build !windows

package manager

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/majidkorai/gollama/pkg/model"
)

// TestStartAutoAssignsPastRunningInstance (P2-T2): with an instance already
// running on 8081 (as recovered from another gollama process, which advances
// nextPort past it) and port 0 (auto-assign), Start must not error and must
// land on the next port. This is what lets 'gollama run' start a second
// instance instead of silently no-op'ing (the old main.go guard) or
// colliding.
func TestStartAutoAssignsPastRunningInstance(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	binDir := filepath.Join(home, ".gollama", "bin")
	modelsDir := filepath.Join(home, ".gollama", "models")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(modelsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "llama-server"), []byte("#!/bin/sh\nsleep 30\n"), 0755); err != nil {
		t.Fatal(err)
	}
	blob := filepath.Join(modelsDir, "fake.gguf")
	if err := os.WriteFile(blob, []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}
	model.SaveIndex(map[string]model.ModelInfo{
		"fake-1b": {Name: "fake-1b", ShortName: "fake-1b", BlobPath: blob, Size: 4},
	})
	model.SaveConfig(&model.Config{
		DefaultFlags: []string{"--ctx-size", "512"},
		Profiles:     map[string]model.Profile{},
		IdleTTL:      0,
	})

	// State as left by recoverOrphans: a running instance on 8081 and
	// nextPort advanced past it.
	m := &Manager{
		instances: map[int]*Instance{
			8081: {Port: 8081, Status: "running", PID: os.Getpid(), Model: "fake-1b"},
		},
		nextPort: 8082,
	}
	inst, err := m.Start("fake-1b", 0, nil, false, nil)
	if err != nil {
		t.Fatalf("Start with auto-assign: %v", err)
	}
	t.Cleanup(func() { m.Stop(inst.Port) })
	if inst.Port == 8081 {
		t.Fatalf("auto-assigned port = 8081, collides with the running instance")
	}
	if inst.Port != 8082 {
		t.Errorf("auto-assigned port = %d, want 8082", inst.Port)
	}
}

// TestRecoverOrphansFindsScriptInstance (P2-T2): a gollama-started process
// whose ps line shows the interpreter first (shebang scripts on macOS) must
// still be recovered — the old pgrep-by-name approach missed it, so a second
// 'gollama run' would collide on the same port.
func TestRecoverOrphansFindsScriptInstance(t *testing.T) {
	scriptDir := t.TempDir()
	script := filepath.Join(scriptDir, "llama-server")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nsleep 30\n"), 0755); err != nil {
		t.Fatal(err)
	}
	proc := exec.Command(script, "-m", "/tmp/fake-model.gguf", "--host", "0.0.0.0", "--port", "18473")
	if err := proc.Start(); err != nil {
		t.Fatalf("starting dummy process: %v", err)
	}
	t.Cleanup(func() { proc.Process.Kill() })

	m := &Manager{instances: make(map[int]*Instance), nextPort: 8081}
	// ps can lag behind process start — retry briefly.
	var found *Instance
	for i := 0; i < 20; i++ {
		m.recoverOrphans()
		if found, _ = m.instances[18473]; found != nil {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if found == nil {
		t.Fatalf("orphan on port 18473 was not recovered; instances: %v", m.List())
	}
	if found.Model != "fake-model.gguf" {
		t.Errorf("recovered model = %q, want %q", found.Model, "fake-model.gguf")
	}
	if found.PID != proc.Process.Pid {
		t.Errorf("recovered PID = %d, want %d", found.PID, proc.Process.Pid)
	}
	if m.nextPort <= 18473 {
		t.Errorf("nextPort = %d, want > 18473 so auto-assign skips the recovered port", m.nextPort)
	}
}
