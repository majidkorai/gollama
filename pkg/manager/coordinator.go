package manager

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/majidkorai/gollama/pkg/model"
)

// SwitchMode selects the switching semantics of a SwitchAndStart call.
type SwitchMode int

const (
	// SwitchText is the text auto-launch semantics: stop other text
	// instances, wait out an in-flight image generation (up to 60s), then a
	// 2s GPU cooldown before the model is (re)started.
	SwitchText SwitchMode = iota
	// SwitchImage is the image-generation semantics: defer with ErrBusy
	// while a text instance has been active in the last 30s, stop idle text
	// otherwise, then start the image profile if none is running.
	SwitchImage
	// SwitchExplicit is an explicit launch (UI quick launch, CLI run/chat):
	// the start is serialized, but no other instance is ever stopped and no
	// existing instance is reused — the caller asked for a start.
	SwitchExplicit
)

func (m SwitchMode) String() string {
	switch m {
	case SwitchText:
		return "text"
	case SwitchImage:
		return "image"
	case SwitchExplicit:
		return "explicit"
	}
	return "unknown"
}

// ErrBusy is returned by a SwitchImage call while a text instance is still
// active. Callers should surface it as 503 + Retry-After: BusyRetryAfter.
// The concrete error is a *BusyError (matches via errors.Is) whose message
// names the busy model, e.g. `text model "foo" is busy`.
var ErrBusy = errors.New("text model is busy")

// BusyError is the concrete form of ErrBusy.
type BusyError struct {
	Model string
}

func (e *BusyError) Error() string { return "text model " + strconv.Quote(e.Model) + " is busy" }

func (e *BusyError) Is(target error) bool { return target == ErrBusy }

// BusyRetryAfter is how many seconds a caller should wait before retrying a
// switch that was deferred with ErrBusy.
const BusyRetryAfter = 30

// ErrSwitchAborted is returned when the caller's abort condition fired while
// waiting for the model to become ready (typically the client disconnecting).
var ErrSwitchAborted = errors.New("client disconnected while model was loading")

// ErrModelExited is returned when the launched process exits before serving.
var ErrModelExited = errors.New("model process exited before becoming ready")

// ErrNotReady is returned when the model did not become ready before the
// load deadline (model.LoadTimeout).
var ErrNotReady = errors.New("model did not become ready")

// SwitchRequest describes one model switch through the Coordinator.
type SwitchRequest struct {
	Model        string
	Mode         SwitchMode
	Port         int // explicit mode only; 0 = auto-assign
	Flags        []string
	ReplaceFlags bool
	Env          map[string]string
	BinaryPath   string
	Profile      string // applied via SetProfile after a fresh start

	// WaitReady holds the switch until the model actually serves (health
	// check passes or it fails), not just until the process has spawned.
	// The proxy path uses this so concurrent different-model requests queue
	// behind a loading model instead of preempting it mid-load.
	WaitReady bool
	// Heartbeat is called while waiting for readiness (throttling is the
	// caller's business — e.g. SSE comment every ~10s).
	Heartbeat func()
	// ShouldAbort aborts the readiness wait when true (client gone).
	ShouldAbort func() bool
}

type inflight struct {
	done chan struct{}
	inst *Instance
	err  error
}

// Coordinator serializes model switches (P3-T1). One switch runs at a time:
//
//   - concurrent requests for the SAME model coalesce — the first caller
//     performs the switch, the others wait on its result and reuse the
//     instance;
//   - requests for DIFFERENT models queue on the switch lock instead of
//     thrashing the GPU with stop/start loops;
//   - a switch that has to wait more than 90s logs loudly.
//
// switchMu is held for the whole switch, including the image grace wait and
// (when WaitReady is set) the readiness wait. mu only guards the in-flight
// map, so a same-model caller can OBSERVE the in-flight entry (and coalesce)
// while the switcher holds switchMu — holding one lock for both purposes
// would make the map invisible to waiting callers.
type Coordinator struct {
	mu       sync.Mutex
	switchMu sync.Mutex
	mgr      *Manager
	inFlight map[string]*inflight
}

// NewCoordinator returns a Coordinator for the given manager.
func NewCoordinator(mgr *Manager) *Coordinator {
	return &Coordinator{mgr: mgr, inFlight: make(map[string]*inflight)}
}

// SwitchAndStart performs one model switch. See the Coordinator doc comment
// for the queueing/coalescing semantics.
func (c *Coordinator) SwitchAndStart(req SwitchRequest) (*Instance, error) {
	start := time.Now()
	if req.Mode != SwitchExplicit {
		c.mu.Lock()
		if inf, ok := c.inFlight[req.Model]; ok {
			// Same model is switching: coalesce. Wait on its done channel
			// (holding no locks) and reuse the result.
			c.mu.Unlock()
			<-inf.done
			if waited := time.Since(start); waited > 90*time.Second {
				slog.Warn("model switch waited unusually long (coalesced on an in-flight switch)", "model", req.Model, "waited", waited.Round(time.Second))
			}
			return c.reuseInflight(req, inf)
		}
		inf := &inflight{done: make(chan struct{})}
		c.inFlight[req.Model] = inf
		c.mu.Unlock()
		inst, err := c.runSwitch(req)
		c.mu.Lock()
		inf.inst, inf.err = inst, err
		delete(c.inFlight, req.Model)
		c.mu.Unlock()
		close(inf.done)
		return inst, err
	}
	// Explicit launches: serialize, never coalesce, never stop others.
	return c.runSwitch(req)
}

// runSwitch takes the switch slot (queueing behind any other model's switch
// — including its grace wait and readiness wait) and performs the switch.
func (c *Coordinator) runSwitch(req SwitchRequest) (inst *Instance, err error) {
	start := time.Now()
	c.switchMu.Lock()
	inst, err = c.perform(req)
	c.switchMu.Unlock()
	if waited := time.Since(start); waited > 90*time.Second {
		slog.Warn("model switch waited unusually long (queued behind another model switch)", "model", req.Model, "waited", waited.Round(time.Second))
	}
	return inst, err
}

// reuseInflight reuses the outcome of the in-flight same-model switch.
func (c *Coordinator) reuseInflight(req SwitchRequest, inf *inflight) (*Instance, error) {
	if inf.err == nil {
		return inf.inst, nil
	}
	// The switch failed (or the first caller aborted its wait). If the
	// model process is still alive, wait for readiness with our OWN abort
	// condition — the first caller's disconnect is not ours. Otherwise
	// propagate the failure.
	if inst := c.mgr.FindInstanceByModel(req.Model); inst != nil && !c.mgr.ProcessExited(inst) {
		if req.WaitReady {
			if err := c.waitReady(inst, req.Heartbeat, req.ShouldAbort); err != nil {
				return nil, err
			}
		}
		return inst, nil
	}
	return nil, inf.err
}

// perform runs one switch. The switch slot (switchMu) is held for the whole
// call, which is what serializes different models (including the up-to-60s
// image grace wait and the readiness wait).
func (c *Coordinator) perform(req SwitchRequest) (*Instance, error) {
	switch req.Mode {
	case SwitchText:
		c.stopOtherText(req.Model)
	case SwitchImage:
		if err := c.stopTextForImage(); err != nil {
			return nil, err
		}
	}

	inst := c.mgr.FindInstanceByModel(req.Model)
	if inst == nil {
		var err error
		if req.Mode == SwitchImage {
			inst, err = c.mgr.StartImage(req.Model, nextImagePort(c.mgr), req.Env)
		} else {
			inst, err = c.mgr.Start(req.Model, req.Port, req.Flags, req.ReplaceFlags, req.Env, req.BinaryPath)
		}
		if err != nil {
			return nil, err
		}
		slog.Info("model switch starting", "model", req.Model, "mode", req.Mode, "port", inst.Port)
		if req.Profile != "" {
			c.mgr.SetProfile(inst.Port, req.Profile)
		}
	}
	if req.WaitReady {
		// Always health-check (even when Ready is already set): the flag is
		// set once at startup and can go stale if the process dies later —
		// a single /health round-trip catches that and fail-fasts on a dead
		// process instead of proxying into a dead port.
		if err := c.waitReady(inst, req.Heartbeat, req.ShouldAbort); err != nil {
			return nil, err
		}
	}
	return inst, nil
}

// stopOtherText is the text-switch eviction: stop other text instances,
// wait out an in-flight image generation (up to 60s, then preempt), and
// cooldown 2s for CUDA to release VRAM. Moved verbatim from the old
// Server.switchToModel — the point of the coordinator is that this blocking
// work now runs under the switch lock.
func (c *Coordinator) stopOtherText(modelName string) {
	var stoppedText bool
	for _, inst := range c.mgr.List() {
		if inst.Status != "running" {
			continue
		}
		if inst.Type != "image" {
			if inst.Model != modelName {
				slog.Info("stopping existing text instance for new model", "model", inst.Model, "port", inst.Port, "newModel", modelName)
				c.mgr.Stop(inst.Port)
				stoppedText = true
			}
			continue
		}
		sinceStart := time.Since(inst.StartedAt)
		sinceActivity := time.Since(inst.LastActivity)
		if sinceStart < 30*time.Second || sinceActivity < 10*time.Second {
			slog.Info("deferring text until image finishes", "model", inst.Model, "port", inst.Port, "runningFor", sinceStart.Round(time.Second))
			deadline := time.Now().Add(60 * time.Second)
			for time.Now().Before(deadline) {
				if !c.mgr.HasInstance(inst.Port) {
					slog.Info("image instance completed, proceeding with text model", "model", modelName)
					break
				}
				time.Sleep(1 * time.Second)
			}
			if c.mgr.HasInstance(inst.Port) {
				slog.Warn("image instance timed out, forcefully preempting")
				c.mgr.Stop(inst.Port)
			}
		} else {
			slog.Info("stopping stale image instance for new model", "model", inst.Model, "port", inst.Port, "newModel", modelName)
			c.mgr.Stop(inst.Port)
		}
	}
	if stoppedText {
		slog.Info("waiting 2s for GPU memory release")
		time.Sleep(2 * time.Second)
	}
}

// stopTextForImage is the image-switch rule: defer (ErrBusy) while a text
// instance has been active within the last 30s, stop idle text otherwise.
func (c *Coordinator) stopTextForImage() error {
	for _, inst := range c.mgr.List() {
		if inst.Status != "running" || inst.Type == "image" {
			continue
		}
		sinceActivity := time.Since(inst.LastActivity)
		if sinceActivity < time.Duration(BusyRetryAfter)*time.Second && sinceActivity >= 0 {
			return &BusyError{Model: inst.Model}
		}
		slog.Info("stopping idle text instance for image generation", "model", inst.Model, "port", inst.Port)
		c.mgr.Stop(inst.Port)
	}
	return nil
}

func nextImagePort(m *Manager) int {
	startPort := 9081
	for _, existing := range m.List() {
		if existing.Type == "image" && existing.Port >= startPort {
			startPort = existing.Port + 1
		}
	}
	return startPort
}

// waitReady blocks until the instance serves /health, mirroring the old
// server-side waitForReady: fail fast when the process exits, honor the
// abort condition, beat the heartbeat, and give up at the load deadline.
func (c *Coordinator) waitReady(inst *Instance, heartbeat func(), shouldAbort func() bool) error {
	healthClient := &http.Client{Timeout: 2 * time.Second}
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", inst.Port)
	deadline := time.Now().Add(model.LoadTimeout())
	for {
		if shouldAbort != nil && shouldAbort() {
			return ErrSwitchAborted
		}
		if resp, err := healthClient.Get(baseURL + "/health"); err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		} else if c.mgr.ProcessExited(inst) {
			return fmt.Errorf("%w: %s", ErrModelExited, logTail(inst.Port, 5))
		}
		if heartbeat != nil {
			heartbeat()
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("%w within %s: %s", ErrNotReady, model.LoadTimeout().Round(time.Second), logTail(inst.Port, 5))
		}
		time.Sleep(500 * time.Millisecond)
	}
}

// logTail returns the last n non-empty lines of an instance's log, used to
// explain why a model failed to start.
func logTail(port int, n int) string {
	data, err := os.ReadFile(filepath.Join(model.GollamaDir(), "logs", fmt.Sprintf("port-%d.log", port)))
	if err != nil {
		return "no log available"
	}
	lines := strings.Split(string(data), "\n")
	out := make([]string, 0, n)
	for i := len(lines) - 1; i >= 0 && len(out) < n; i-- {
		if line := strings.TrimSpace(lines[i]); line != "" {
			out = append(out, line)
		}
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return strings.Join(out, " | ")
}

// ProcessExited reports whether the instance's process is gone. For
// gollama-launched processes the WaitDone channel (closed once the process
// is fully reaped) is authoritative; recovered orphans fall back to a
// signal-0 probe.
func (m *Manager) ProcessExited(inst *Instance) bool {
	if inst == nil || inst.PID <= 0 {
		return true
	}
	if inst.Cmd != nil && inst.Cmd.Process != nil {
		select {
		case <-inst.WaitDone:
			return true
		default:
		}
		return processGone(inst.Cmd.Process)
	}
	proc, err := os.FindProcess(inst.PID)
	if err != nil {
		return true
	}
	return processGone(proc)
}

// processGone reports whether the process no longer exists. ESRCH (a
// foreign PID that is gone) and os.ErrProcessDone (our own already-reaped
// child, where the kernel answers ECHILD) mean gone; EPERM means it exists
// but we may not signal it (e.g. PID 1), which must count as alive.
func processGone(proc *os.Process) bool {
	err := proc.Signal(syscall.Signal(0))
	return errors.Is(err, os.ErrProcessDone) || errors.Is(err, syscall.ESRCH)
}
