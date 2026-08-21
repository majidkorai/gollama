package manager

import (
	"os"
	"strings"
	"testing"
)

func TestNewManager(t *testing.T) {
	m := NewManager()
	if m == nil {
		t.Fatal("NewManager() returned nil")
	}
}

func TestList(t *testing.T) {
	m := NewManager()
	list := m.List()
	if list == nil {
		t.Fatal("List() returned nil")
	}
}

func TestStopNonExistent(t *testing.T) {
	m := NewManager()
	err := m.Stop(99999)
	if err == nil {
		t.Error("expected error when stopping non-existent instance")
	}
}

func TestUpdateTokensNonExistent(t *testing.T) {
	m := NewManager()
	m.UpdateTokens(99999, 42.5)
}

func TestUpdateTokens(t *testing.T) {
	m := NewManager()
	m.UpdateTokens(0, 10.5)
}

func TestNewManagerMultiple(t *testing.T) {
	m1 := NewManager()
	m2 := NewManager()
	if m1 == nil || m2 == nil {
		t.Fatal("NewManager() returned nil")
	}
}

func TestIsLlamaServerCommandLine(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"llama-server -m /root/models/m.gguf --host 0.0.0.0 --port 8081", true},
		{"/usr/local/bin/llama-server -m m.gguf --host 0.0.0.0 --port 8081", true},
		// shebang script: interpreter first, script path second (macOS ps form)
		{"/bin/sh /root/.gollama/bin/llama-server --host 127.0.0.1 --port 8081", true},
		{"llama-server --port 8081", false},                // no --host: not gollama-started
		{"/usr/bin/llama-bench --host 0.0.0.0", false},     // different binary
		{"llama-server --host=0.0.0.0 --port 8081", false}, // gollama always passes --host with a value arg
		{"bash -c sleep 30", false},
	}
	for _, tt := range tests {
		if got := isLlamaServerCommandLine(tt.in); got != tt.want {
			t.Errorf("isLlamaServerCommandLine(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestStartRejectsRunningSlot(t *testing.T) {
	m := &Manager{
		instances: map[int]*Instance{
			8124: {Port: 8124, Status: "running", PID: os.Getpid()},
		},
		nextPort: 9000,
	}
	_, err := m.Start("fake-model", 8124, nil, false, nil)
	if err == nil || !strings.Contains(err.Error(), "already in use") {
		t.Fatalf("err = %v, want 'already in use' for running slot", err)
	}
}
