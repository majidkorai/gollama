//go:build !windows

package manager

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
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
		reserved: make(map[int]bool),
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

	m := &Manager{instances: make(map[int]*Instance), nextPort: 8081, reserved: make(map[int]bool)}
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

// dummyHome sets up an isolated HOME with a dummy llama-server script (the
// given shebang body) and a "fake-1b" index entry, and returns a fresh
// Manager (no background goroutines).
func dummyHome(t *testing.T, script string) *Manager {
	t.Helper()
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
	if err := os.WriteFile(filepath.Join(binDir, "llama-server"), []byte(script), 0755); err != nil {
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
	return &Manager{instances: make(map[int]*Instance), nextPort: 8081, reserved: make(map[int]bool)}
}

// TestStartParallelNoSamePortNoSerialization (P3-T2): two auto-assign
// Starts fired in parallel must (a) never be handed the same port, and
// (b) not serialize behind the port-availability retry sleep. With the
// reservation map the 15×200ms retry (see allocateAvailablePort) runs
// OUTSIDE m.mu; before P3-T2 it ran under the lock, so two Starts each
// forced through a busy candidate would take ~2× the retry window. We block
// both auto-assign candidates (8081, 8082) so every Start pays the full
// retry window, then assert the pair finishes in well under two windows.
func TestStartParallelNoSamePortNoSerialization(t *testing.T) {
	m := dummyHome(t, "#!/bin/sh\nsleep 30\n")
	// Keep the background health-poll goroutines from lingering for the
	// default 5m load timeout.
	t.Setenv("GOLLAMA_MODEL_LOAD_TIMEOUT", "2")

	// Occupy both auto-assign candidates so each Start hits the busy-port
	// retry loop. Skip (not fail) if the ports are already taken.
	var blockers []net.Listener
	for _, p := range []string{"127.0.0.1:8081", "127.0.0.1:8082"} {
		ln, err := net.Listen("tcp", p)
		if err != nil {
			for _, b := range blockers {
				b.Close()
			}
			t.Skipf("candidate port busy in this environment (%s): %v", p, err)
		}
		blockers = append(blockers, ln)
	}
	defer func() {
		for _, b := range blockers {
			b.Close()
		}
	}()

	start := time.Now()
	resCh := make(chan struct {
		inst *Instance
		err  error
	}, 2)
	for i := 0; i < 2; i++ {
		go func() {
			inst, err := m.Start("fake-1b", 0, nil, false, nil)
			resCh <- struct {
				inst *Instance
				err  error
			}{inst, err}
		}()
	}
	ports := make([]int, 0, 2)
	for i := 0; i < 2; i++ {
		r := <-resCh
		if r.err != nil {
			t.Fatalf("parallel Start %d: %v", i, r.err)
		}
		ports = append(ports, r.inst.Port)
		port := r.inst.Port
		t.Cleanup(func() { m.Stop(port) })
	}
	elapsed := time.Since(start)

	// (a) never the same port.
	if ports[0] == ports[1] {
		t.Fatalf("parallel Starts were handed the same port %d", ports[0])
	}

	// (b) not serialized on the retry sleep. One busy candidate costs one
	// retry window (15×200ms ≈ 3s); if the retry ran under the lock the two
	// Starts would take ~2 windows (≈6s). Assert comfortably under that.
	const retryWindow = 3 * time.Second
	if elapsed >= 2*retryWindow-500*time.Millisecond {
		t.Fatalf("parallel Starts serialized on the port-retry sleep: %v (two windows ≈ %v)", elapsed, 2*retryWindow)
	}
	t.Logf("two parallel blocked Starts: ports %v in %v", ports, elapsed)
}

// waitForExit blocks until the instance's process has fully exited
// (WaitDone closed), failing after the timeout.
func waitForExit(t *testing.T, inst *Instance, timeout time.Duration) {
	t.Helper()
	if inst.WaitDone == nil {
		t.Fatal("instance has no WaitDone")
	}
	select {
	case <-inst.WaitDone:
	case <-time.After(timeout):
		t.Fatalf("process did not exit within %v", timeout)
	}
}

// TestStopEscalatesToKillWhenSIGINTIgnored (P2-T3): a process that ignores
// SIGINT must be SIGKILL'ed after the 500ms grace period, and the instance
// must be removed.
func TestStopEscalatesToKillWhenSIGINTIgnored(t *testing.T) {
	// The marker file is written by the script right after its trap line,
	// so its presence proves the trap is in place.
	m := dummyHome(t, "#!/bin/sh\ntrap '' INT\necho started >> $HOME/dbg.log\nwhile :; do sleep 1; done\n")
	inst, err := m.Start("fake-1b", 0, nil, false, nil)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Wait for the process to actually start running the script (marker
	// file). Some sandboxed environments block child processes in dyld
	// startup (waiting on a process-monitoring notification), where a
	// SIGINT kills them before any trap is in place — skip there.
	marker := filepath.Join(os.Getenv("HOME"), "dbg.log")
	markerSeen := false
	for i := 0; i < 100; i++ {
		if _, err := os.Stat(marker); err == nil {
			markerSeen = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !markerSeen {
		t.Skip("child process did not start running its script (sandboxed dyld block) — cannot test SIGINT escalation")
	}

	start := time.Now()
	if err := m.Stop(inst.Port); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	elapsed := time.Since(start)
	if elapsed < 400*time.Millisecond {
		t.Errorf("Stop returned in %v — the SIGINT grace period was skipped", elapsed)
	}
	if elapsed > 5*time.Second {
		t.Errorf("Stop took %v — SIGKILL escalation did not bound the wait", elapsed)
	}
	if proc, _ := os.FindProcess(inst.PID); proc != nil && proc.Signal(syscall.Signal(0)) == nil {
		t.Errorf("process %d still alive after Stop", inst.PID)
	}
	if m.HasInstance(inst.Port) {
		t.Errorf("instance still in map after Stop")
	}
}

// TestStopFastPathWhenAlreadyExited (P2-T3): stopping an instance whose
// process already exited must not pay the 500ms grace wait.
func TestStopFastPathWhenAlreadyExited(t *testing.T) {
	m := dummyHome(t, "#!/bin/sh\nexit 0\n")
	inst, err := m.Start("fake-1b", 0, nil, false, nil)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	waitForExit(t, inst, 5*time.Second)

	start := time.Now()
	if err := m.Stop(inst.Port); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 400*time.Millisecond {
		t.Errorf("Stop took %v for an already-exited process, want < 400ms", elapsed)
	}
}

// TestStopAll (P2-T3): StopAll stops every instance in parallel and returns
// their ports; the map ends up empty.
func TestStopAll(t *testing.T) {
	m := dummyHome(t, "#!/bin/sh\nsleep 30\n")
	inst1, err := m.Start("fake-1b", 0, nil, false, nil)
	if err != nil {
		t.Fatalf("Start 1: %v", err)
	}
	inst2, err := m.Start("fake-1b", 0, nil, false, nil)
	if err != nil {
		t.Fatalf("Start 2: %v", err)
	}
	time.Sleep(300 * time.Millisecond)

	stopped := m.StopAll()
	if len(stopped) != 2 {
		t.Fatalf("StopAll stopped %d instances, want 2: %v", len(stopped), stopped)
	}
	got := map[int]bool{}
	for _, p := range stopped {
		got[p] = true
	}
	if !got[inst1.Port] || !got[inst2.Port] {
		t.Errorf("StopAll ports = %v, want [%d %d]", stopped, inst1.Port, inst2.Port)
	}
	if len(m.List()) != 0 {
		t.Errorf("instances remaining after StopAll: %v", m.List())
	}
	for _, inst := range []*Instance{inst1, inst2} {
		if proc, _ := os.FindProcess(inst.PID); proc != nil && proc.Signal(syscall.Signal(0)) == nil {
			t.Errorf("process %d still alive after StopAll", inst.PID)
		}
	}
}

// TestConfirmOrphanReady (P3-T3): a recovered orphan is registered
// optimistically, but the confirming health poll must (a) confirm readiness
// and snapshot the one-shot metrics when the process actually serves /health,
// and (b) mark the instance stopped when the process dies before serving.
func TestConfirmOrphanReady(t *testing.T) {
	t.Setenv("GOLLAMA_MODEL_LOAD_TIMEOUT", "5")

	m := &Manager{instances: make(map[int]*Instance), nextPort: 8081, reserved: make(map[int]bool)}
	// Locked reads so the test never races the confirming poll's writes.
	memOf := func(inst *Instance) float64 {
		m.mu.Lock()
		defer m.mu.Unlock()
		return inst.MemoryMB
	}
	readyOf := func(inst *Instance) bool {
		m.mu.Lock()
		defer m.mu.Unlock()
		return inst.Ready
	}
	statusOf := func(inst *Instance) string {
		m.mu.Lock()
		defer m.mu.Unlock()
		return inst.Status
	}
	str := func(inst *Instance) string {
		m.mu.Lock()
		defer m.mu.Unlock()
		return fmt.Sprintf("ready=%v mem=%.1f status=%s", inst.Ready, inst.MemoryMB, inst.Status)
	}

	// (a) live process serving /health -> Ready confirmed + metrics snapshotted.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	livePort := srv.Listener.Addr().(*net.TCPAddr).Port
	live := &Instance{Port: livePort, PID: os.Getpid(), Status: "running", Ready: true}
	m.instances[livePort] = live
	m.confirmOrphanReady(live)
	if !waitForCond(t, 5*time.Second, func() bool { return memOf(live) > 0 }) {
		t.Fatalf("confirming poll did not snapshot metrics: %s", str(live))
	}
	if !readyOf(live) {
		t.Fatalf("confirmed orphan not Ready: %s", str(live))
	}

	// (b) process dies before serving -> instance marked stopped.
	cmd := exec.Command("/bin/sh", "-c", "exit 0")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}
	deadPort := 19999 // nothing listens here
	dead := &Instance{Port: deadPort, PID: cmd.Process.Pid, Status: "running", Ready: true}
	m.instances[deadPort] = dead
	m.confirmOrphanReady(dead)
	if !waitForCond(t, 5*time.Second, func() bool { return statusOf(dead) == "stopped" }) {
		t.Fatalf("dead orphan not marked stopped: %s", str(dead))
	}
}

// waitForCond polls cond every 20ms until it is true or the timeout elapses,
// returning whether the condition was met.
func waitForCond(t *testing.T, timeout time.Duration, cond func() bool) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return cond()
}
