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
		reserved: make(map[int]bool),
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

// TestRegisterOrphanFromCommandLine (P2-T6): only command lines that look
// gollama-launched (carry the --host flag) are recovered; anything else is
// skipped so no phantom instances or burned ports. Cross-platform: the
// parser is pure, so it runs on every OS.
func TestRegisterOrphanFromCommandLine(t *testing.T) {
	fresh := func() *Manager {
		return &Manager{instances: map[int]*Instance{}, nextPort: 8081, reserved: make(map[int]bool)}
	}

	// Not a gollama process (no --host) -> skipped, no state change.
	m := fresh()
	if m.registerOrphanFromCommandLine(111, `C:\tools\llama-bench.exe -m C:\models\foo.gguf`) {
		t.Fatal("non-gollama command line was registered")
	}
	if len(m.instances) != 0 || m.nextPort != 8081 {
		t.Fatalf("phantom state: instances=%v nextPort=%d", m.instances, m.nextPort)
	}

	// Gollama process with an explicit port -> registered on that port,
	// nextPort advanced past it, model name reduced to its basename.
	m = fresh()
	if !m.registerOrphanFromCommandLine(222, `"C:\.gollama\bin\llama-server.exe" -m C:\.gollama\models\foo.gguf --host 0.0.0.0 --port 8085`) {
		t.Fatal("gollama command line was not registered")
	}
	inst, ok := m.instances[8085]
	if !ok || inst.PID != 222 || inst.Model != "foo.gguf" {
		t.Fatalf("instance = %+v, want port 8085 pid 222 model foo.gguf", inst)
	}
	if m.nextPort != 8086 {
		t.Fatalf("nextPort = %d, want 8086", m.nextPort)
	}

	// Same PID twice -> no duplicate registration.
	if m.registerOrphanFromCommandLine(222, `llama-server -m x.gguf --host 1.2.3.4 --port 8099`) {
		t.Fatal("duplicate PID was registered")
	}

	// Gollama process without --port -> guessed port from nextPort.
	m = fresh()
	if !m.registerOrphanFromCommandLine(333, `llama-server -m x.gguf --host 1.2.3.4`) {
		t.Fatal("gollama command line without --port was not registered")
	}
	if got, ok := m.instances[8081]; !ok || got.PID != 333 {
		t.Fatalf("guessed port 8081 not registered correctly: %v", m.instances)
	}
	if m.nextPort != 8082 {
		t.Fatalf("nextPort = %d, want 8082", m.nextPort)
	}
}

// TestRecoverOrphanWindowsSkipsSelf (P2-T6, P3-T2): the test process's own
// command line is not a gollama one, so nothing may be registered — on
// Windows via the no---host rule, on other OSes because wmic is missing and
// the PID is skipped instead of guessed. Either way no phantom, no burned
// port.
func TestRecoverOrphanWindowsSkipsSelf(t *testing.T) {
	m := &Manager{instances: map[int]*Instance{}, nextPort: 8081, reserved: map[int]bool{}}
	// Mirror recoverOrphansWindows for a single PID: fetch the command line
	// (absent on non-Windows, where wmic is missing) and, if present, attempt
	// to register it. The test binary's own process is not gollama-launched,
	// so nothing may be registered and no port may be burned.
	if cmdLine, ok := m.windowsCommandLine(os.Getpid()); ok {
		m.registerOrphanFromCommandLine(os.Getpid(), cmdLine)
	}
	if len(m.instances) != 0 {
		t.Fatalf("phantom instance registered: %v", m.instances)
	}
	if m.nextPort != 8081 {
		t.Fatalf("nextPort burned: %d", m.nextPort)
	}
}

// TestParseProcStatTicks (P2-T7): utime/stime extraction from a
// /proc/<pid>/stat line. comm (field 2) is wrapped in parentheses and may
// itself contain spaces and parentheses, so the parser must start after the
// last ')'.
func TestParseProcStatTicks(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want float64
		ok   bool
	}{
		{"plain", "1 (bash) S 0 1 1 0 -1 4194560 100 0 0 0 1234 567 0 0 20 0 1 0 3456 12345678 90 18446744073709551615 134217728 134217728 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0", 1234 + 567, true},
		{"comm with spaces and parens", "42 (my proc (with) spaces) S 0 1 1 0 -1 0 0 0 0 0 10 20 0 0 20 0 1 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0", 30, true},
		{"no closing paren", "1 (bash S 0 1 1 0 -1 0 0 0 0 0 10 20", 0, false},
		{"too few fields", "1 (bash) S 0 1 1 0 -1 0 0 0 0 0 10", 0, false},
		{"non-numeric utime", "1 (bash) S 0 1 1 0 -1 0 0 0 0 0 x 20", 0, false},
	}
	for _, c := range cases {
		got, ok := parseProcStatTicks([]byte(c.in))
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("%s: got (%v, %v), want (%v, %v)", c.name, got, ok, c.want, c.ok)
		}
	}
}

// TestParseGpuUtilCSV (P2-T7): one entry per GPU line; unparseable lines
// are skipped; no valid lines -> not ok.
func TestParseGpuUtilCSV(t *testing.T) {
	per, ok := parseGpuUtilCSV([]byte("92\n3\n"))
	if !ok || len(per) != 2 || per[0] != 92 || per[1] != 3 {
		t.Fatalf("multi = (%v, %v), want ([92 3], true)", per, ok)
	}
	per, ok = parseGpuUtilCSV([]byte("17\n"))
	if !ok || len(per) != 1 || per[0] != 17 {
		t.Fatalf("single = (%v, %v), want ([17], true)", per, ok)
	}
	if _, ok := parseGpuUtilCSV([]byte("garbage\n")); ok {
		t.Fatal("garbage lines parsed")
	}
	if _, ok := parseGpuUtilCSV(nil); ok {
		t.Fatal("empty output parsed")
	}
}

// TestSetGpuUtil (P2-T7): GpuUtil is the max of the current sample (not a
// running high-water mark) and GpuUtilPerGPU keeps the breakdown.
func TestSetGpuUtil(t *testing.T) {
	inst := &Instance{Port: 8081}
	setGpuUtil(inst, []float64{3, 92})
	if inst.GpuUtil != 92 || len(inst.GpuUtilPerGPU) != 2 || inst.GpuUtilPerGPU[0] != 3 || inst.GpuUtilPerGPU[1] != 92 {
		t.Fatalf("after [3 92]: GpuUtil=%v per=%v, want 92 / [3 92]", inst.GpuUtil, inst.GpuUtilPerGPU)
	}
	setGpuUtil(inst, []float64{50, 10})
	if inst.GpuUtil != 50 {
		t.Fatalf("after [50 10]: GpuUtil=%v, want 50 (max of the new sample)", inst.GpuUtil)
	}
	setGpuUtil(inst, []float64{})
	if inst.GpuUtil != 50 || len(inst.GpuUtilPerGPU) != 2 {
		t.Fatalf("empty sample changed state: GpuUtil=%v per=%v", inst.GpuUtil, inst.GpuUtilPerGPU)
	}
}

func TestStartRejectsRunningSlot(t *testing.T) {
	m := &Manager{
		instances: map[int]*Instance{
			8124: {Port: 8124, Status: "running", PID: os.Getpid()},
		},
		nextPort: 9000,
		reserved: make(map[int]bool),
	}
	_, err := m.Start("fake-model", 8124, nil, false, nil)
	if err == nil || !strings.Contains(err.Error(), "already in use") {
		t.Fatalf("err = %v, want 'already in use' for running slot", err)
	}
}
