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

// TestFindInstanceByModelDeterministic (P2-T5): matching must pick the same
// instance regardless of map iteration order. Tier order is
// exact/same-blob > short-name > suffix > substring, ties broken by lowest
// port. The old code returned whichever candidate the map happened to yield
// first.
func TestFindInstanceByModelDeterministic(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	m := &Manager{
		instances: map[int]*Instance{
			8090: {Port: 8090, PID: os.Getpid(), Model: "hf.co/org/Qwen-99B-GGUF:Q8_0", Status: "running"},
			8081: {Port: 8081, PID: os.Getpid(), Model: "qwen3.6-27b", Status: "running"},
			8082: {Port: 8082, PID: os.Getpid(), Model: "qwen3.6-27b-q4", Status: "running"},
			8083: {Port: 8083, PID: os.Getpid(), Model: "qwen3.6-27b", Status: "stopped"},
		},
		nextPort: 9000,
	}

	// Exact match wins even though other candidates match by suffix/substring.
	if inst := m.FindInstanceByModel("qwen3.6-27b"); inst == nil || inst.Port != 8081 {
		t.Fatalf("exact = %+v, want port 8081 (and not the stopped 8083)", inst)
	}
	// Suffix tier beats substring tier: "27b" is a suffix of 8081's model and
	// a substring of 8082's.
	if inst := m.FindInstanceByModel("27b"); inst == nil || inst.Port != 8081 {
		t.Fatalf("suffix = %+v, want port 8081", inst)
	}
	// All three running instances contain "qwen" by substring only — the tie
	// must go to the lowest port (8081), deterministically.
	for i := 0; i < 50; i++ {
		if inst := m.FindInstanceByModel("qwen"); inst == nil || inst.Port != 8081 {
			t.Fatalf("iteration %d: substring tie = %+v, want port 8081", i, inst)
		}
	}
	// No match at all.
	if inst := m.FindInstanceByModel("does-not-exist"); inst != nil {
		t.Fatalf("expected nil, got %+v", inst)
	}
	// Empty query returns nil instead of matching everything by substring.
	if inst := m.FindInstanceByModel(""); inst != nil {
		t.Fatalf("empty query = %+v, want nil", inst)
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
