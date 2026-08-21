//go:build !windows

package manager

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/majidkorai/gollama/pkg/model"
)

// multiModelHome sets up an isolated HOME with a dummy llama-server script
// (the given shebang body), the given text models in the index, and - when
// image is true - dummy image python/app paths (GOLLAMA_IMAGE_PYTHON=/bin/sh
// plus a sleeping app script) so StartImage can launch without a real
// diffusers stack. Returns a fresh Manager (no background goroutines).
func multiModelHome(t *testing.T, script string, models []string, image bool) *Manager {
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
	idx := make(map[string]model.ModelInfo, len(models))
	for _, name := range models {
		blob := filepath.Join(modelsDir, name+".gguf")
		if err := os.WriteFile(blob, []byte("fake"), 0644); err != nil {
			t.Fatal(err)
		}
		idx[name] = model.ModelInfo{Name: name, ShortName: name, BlobPath: blob, Size: 4}
	}
	model.SaveIndex(idx)
	model.SaveConfig(&model.Config{
		DefaultFlags: []string{"--ctx-size", "512"},
		Profiles:     map[string]model.Profile{},
		IdleTTL:      0,
	})
	if image {
		app := filepath.Join(home, "image-app.sh")
		if err := os.WriteFile(app, []byte("#!/bin/sh\nsleep 30\n"), 0755); err != nil {
			t.Fatal(err)
		}
		t.Setenv("GOLLAMA_IMAGE_PYTHON", "/bin/sh")
		t.Setenv("GOLLAMA_IMAGE_APP", app)
	}
	return &Manager{instances: make(map[int]*Instance), nextPort: 8081, reserved: make(map[int]bool)}
}

// TestCoordinatorCoalescesSameModel (P3-T1): concurrent switches for the
// same model must start exactly ONE process; the other callers wait on the
// in-flight switch and share its outcome instead of each starting their own
// instance.
func TestCoordinatorCoalescesSameModel(t *testing.T) {
	t.Setenv("GOLLAMA_MODEL_LOAD_TIMEOUT", "3")
	m := multiModelHome(t, "#!/bin/sh\nsleep 30\n", []string{"fake-1b"}, false)
	coord := NewCoordinator(m)
	t.Cleanup(func() { m.StopAll() })

	const callers = 3
	errs := make([]error, callers)
	var wg sync.WaitGroup
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = coord.SwitchAndStart(SwitchRequest{
				Model:     "fake-1b",
				Mode:      SwitchText,
				WaitReady: true,
			})
		}(i)
	}
	wg.Wait()

	// The dummy server never passes /health, so every caller must observe
	// the load-timeout failure - none of them may have masked it by starting
	// a second process on its own.
	for i, err := range errs {
		if err == nil {
			t.Fatalf("caller %d: expected error (model never ready), got nil", i)
		}
		if !errors.Is(err, ErrNotReady) {
			t.Errorf("caller %d: err = %v, want ErrNotReady", i, err)
		}
	}
	running := 0
	for _, inst := range m.List() {
		if inst.Model == "fake-1b" && inst.Status == "running" {
			running++
		}
	}
	if running != 1 {
		t.Errorf("running fake-1b instances = %d, want 1 (coalesced): %v", running, m.List())
	}
}

// TestCoordinatorQueuesDifferentModels (P3-T1): switches for different
// models must serialize on the switch slot - each pays the 2s GPU cooldown
// after stopping the previous text instance - instead of racing and
// thrashing the GPU with overlapping stop/start cycles.
func TestCoordinatorQueuesDifferentModels(t *testing.T) {
	m := multiModelHome(t, "#!/bin/sh\nsleep 30\n", []string{"model-a", "model-b", "model-c"}, false)
	coord := NewCoordinator(m)
	t.Cleanup(func() { m.StopAll() })

	instA, err := coord.SwitchAndStart(SwitchRequest{Model: "model-a", Mode: SwitchText})
	if err != nil {
		t.Fatalf("start model-a: %v", err)
	}

	started := time.Now()
	type result struct {
		inst *Instance
		err  error
	}
	resB := make(chan result, 1)
	go func() {
		inst, err := coord.SwitchAndStart(SwitchRequest{Model: "model-b", Mode: SwitchText})
		resB <- result{inst, err}
	}()
	// Let B switch get going before C queues behind it.
	time.Sleep(300 * time.Millisecond)
	instC, err := coord.SwitchAndStart(SwitchRequest{Model: "model-c", Mode: SwitchText})
	if err != nil {
		t.Fatalf("switch model-c: %v", err)
	}
	b := <-resB
	if b.err != nil {
		t.Fatalf("switch model-b: %v", b.err)
	}
	elapsed := time.Since(started)

	// Serialized: each switch stops a text instance and pays the 2s GPU
	// cooldown, so the pair takes at least 2 x 2s. Unserialized (racing)
	// switches would finish in ~2s total.
	if elapsed < 3500*time.Millisecond {
		t.Errorf("two different-model switches took %v - they should serialize (each pays the 2s GPU cooldown)", elapsed)
	}
	if m.HasInstance(instA.Port) {
		t.Errorf("model-a still running after the switches to b/c")
	}
	if m.HasInstance(b.inst.Port) {
		t.Errorf("model-b still running after the switch to c")
	}
	if !m.HasInstance(instC.Port) {
		t.Errorf("model-c not running after the switches")
	}
}

// TestCoordinatorImageBusyThenIdle (P3-T1): an image switch must defer with
// ErrBusy while a text instance has been active in the last 30s (and stop
// nothing), and must stop an idle text instance to free VRAM.
func TestCoordinatorImageBusyThenIdle(t *testing.T) {
	m := multiModelHome(t, "#!/bin/sh\nsleep 30\n", []string{"fake-1b", "flux-fake"}, true)
	coord := NewCoordinator(m)
	t.Cleanup(func() { m.StopAll() })

	inst, err := coord.SwitchAndStart(SwitchRequest{Model: "fake-1b", Mode: SwitchText})
	if err != nil {
		t.Fatalf("start text: %v", err)
	}
	m.TouchActivity(inst.Port)

	// Busy: text active within the last 30s.
	_, err = coord.SwitchAndStart(SwitchRequest{Model: "flux-fake", Mode: SwitchImage})
	if !errors.Is(err, ErrBusy) {
		t.Fatalf("busy switch: err = %v, want ErrBusy", err)
	}
	if !m.HasInstance(inst.Port) {
		t.Errorf("active text instance was stopped despite ErrBusy")
	}

	// Idle: age the last-activity timestamp past the 30s window (the test
	// manager has no background goroutines, so the direct write is safe).
	for _, i := range m.List() {
		if i.Port == inst.Port {
			i.LastActivity = time.Now().Add(-31 * time.Second)
		}
	}
	img, err := coord.SwitchAndStart(SwitchRequest{Model: "flux-fake", Mode: SwitchImage})
	if err != nil {
		t.Fatalf("idle switch: %v", err)
	}
	if m.HasInstance(inst.Port) {
		t.Errorf("idle text instance not stopped for image generation")
	}
	if img == nil || img.Type != "image" {
		t.Errorf("image switch returned %+v, want a running image instance", img)
	}
}

// TestCoordinatorExplicitNeverStops (P3-T1): an explicit launch (UI quick
// launch, CLI run/chat) serializes on the switch slot but never stops or
// reuses other instances, and pays no GPU cooldown.
func TestCoordinatorExplicitNeverStops(t *testing.T) {
	m := multiModelHome(t, "#!/bin/sh\nsleep 30\n", []string{"model-a", "model-b"}, false)
	coord := NewCoordinator(m)
	t.Cleanup(func() { m.StopAll() })

	instA, err := coord.SwitchAndStart(SwitchRequest{Model: "model-a", Mode: SwitchText})
	if err != nil {
		t.Fatalf("start model-a: %v", err)
	}

	started := time.Now()
	instB, err := coord.SwitchAndStart(SwitchRequest{Model: "model-b", Mode: SwitchExplicit})
	if err != nil {
		t.Fatalf("explicit model-b: %v", err)
	}
	elapsed := time.Since(started)

	if !m.HasInstance(instA.Port) {
		t.Errorf("explicit launch stopped an unrelated instance")
	}
	if instB.Port == instA.Port || !m.HasInstance(instB.Port) {
		t.Errorf("explicit launch did not start a new instance (A=%d B=%d)", instA.Port, instB.Port)
	}
	if elapsed > 1500*time.Millisecond {
		t.Errorf("explicit launch took %v - it must not pay the text-switch GPU cooldown", elapsed)
	}
}

// TestProcessExited (P3-T1/T3): gollama-launched processes are tracked via
// the WaitDone channel; recovered orphans (no Cmd) fall back to the
// signal-0 probe.
func TestProcessExited(t *testing.T) {
	m := multiModelHome(t, "#!/bin/sh\nsleep 30\n", []string{"fake-1b"}, false)
	inst, err := m.Start("fake-1b", 0, nil, false, nil)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { m.Stop(inst.Port) })
	if m.ProcessExited(inst) {
		t.Error("ProcessExited = true for a running instance")
	}
	if err := m.Stop(inst.Port); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if !m.ProcessExited(inst) {
		t.Error("ProcessExited = false after Stop")
	}

	// Orphan path (no Cmd): signal-0 probe. A reaped PID must report
	// exited; PID 1 is alive on every Unix system.
	proc := exec.Command("/bin/sh", "-c", "exit 0")
	if err := proc.Start(); err != nil {
		t.Fatalf("spawn: %v", err)
	}
	if err := proc.Wait(); err != nil {
		t.Fatalf("wait: %v", err)
	}
	if !m.ProcessExited(&Instance{PID: proc.Process.Pid}) {
		t.Error("exited orphan process not reported exited")
	}
	if m.ProcessExited(&Instance{PID: 1}) {
		t.Error("PID 1 reported exited")
	}
}
