package server

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/majidkorai/gollama/pkg/manager"
)

// newTestServer returns a Server backed by a manager isolated under a
// temporary HOME so no real config/instances are touched.
func newTestServer(t *testing.T) *Server {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	return New(manager.NewManager(), "8080")
}

// startHealthServer returns the port of a local HTTP server whose /health
// endpoint answers 503 until readyAt (absolute time), then 200. A zero
// readyAt means it never becomes ready.
func startHealthServer(t *testing.T, readyAt time.Time) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go func() {
		http.Serve(ln, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/health" {
				http.NotFound(w, r)
				return
			}
			if readyAt.IsZero() || time.Now().Before(readyAt) {
				w.WriteHeader(http.StatusServiceUnavailable)
				fmt.Fprint(w, `{"status":"loading model"}`)
				return
			}
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"status":"ok"}`)
		}))
	}()
	t.Cleanup(func() { ln.Close() })
	return ln.Addr().(*net.TCPAddr).Port
}

func TestModelLoadTimeoutEnv(t *testing.T) {
	if got := modelLoadTimeout(); got != 5*time.Minute {
		t.Fatalf("default = %v, want 5m", got)
	}
	t.Setenv("GOLLAMA_MODEL_LOAD_TIMEOUT", "42")
	if got := modelLoadTimeout(); got != 42*time.Second {
		t.Fatalf("env override = %v, want 42s", got)
	}
	t.Setenv("GOLLAMA_MODEL_LOAD_TIMEOUT", "garbage")
	if got := modelLoadTimeout(); got != 5*time.Minute {
		t.Fatalf("garbage env = %v, want 5m", got)
	}
}

func TestSseErrorChunk(t *testing.T) {
	chunk := sseErrorChunk(`failed: "quote" and | pipe`)
	if !strings.HasPrefix(string(chunk), "data: ") || !strings.HasSuffix(string(chunk), "\n\n") {
		t.Fatalf("bad SSE framing: %q", chunk)
	}
	var ev struct {
		Error map[string]string `json:"error"`
	}
	if err := json.Unmarshal(chunk[len("data: "):], &ev); err != nil {
		t.Fatalf("chunk is not a valid JSON event: %v", err)
	}
	if ev.Error["type"] != "server_error" {
		t.Fatalf("error type = %q, want server_error", ev.Error["type"])
	}
	if !strings.Contains(ev.Error["message"], `failed: "quote" and | pipe`) {
		t.Fatalf("message not preserved: %q", ev.Error["message"])
	}
}

func TestWaitForReadySuccessAfterDelay(t *testing.T) {
	s := newTestServer(t)
	port := startHealthServer(t, time.Now().Add(1200*time.Millisecond))
	start := time.Now()
	if err := s.waitForReady(port, 10*time.Second, nil, nil); err != nil {
		t.Fatalf("waitForReady = %v", err)
	}
	if elapsed := time.Since(start); elapsed < 1100*time.Millisecond {
		t.Fatalf("returned before server became ready: %v", elapsed)
	}
}

func TestWaitForReadyTimeout(t *testing.T) {
	s := newTestServer(t)
	port := startHealthServer(t, time.Time{}) // never ready
	start := time.Now()
	err := s.waitForReady(port, 2*time.Second, nil, nil)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "did not become ready within 2s") {
		t.Fatalf("unexpected error: %v", err)
	}
	if elapsed := time.Since(start); elapsed < 1900*time.Millisecond {
		t.Fatalf("returned before deadline: %v", elapsed)
	}
}

func TestWaitForReadyAbort(t *testing.T) {
	s := newTestServer(t)
	port := startHealthServer(t, time.Time{})
	start := time.Now()
	err := s.waitForReady(port, 30*time.Second, nil, func() bool { return true })
	if err == nil || !strings.Contains(err.Error(), "client disconnected") {
		t.Fatalf("err = %v, want client disconnected", err)
	}
	if elapsed := time.Now().Sub(start); elapsed > 2*time.Second {
		t.Fatalf("abort did not stop the wait promptly: %v", elapsed)
	}
}

func TestWaitForReadyBeats(t *testing.T) {
	s := newTestServer(t)
	port := startHealthServer(t, time.Now().Add(1200*time.Millisecond))
	beats := 0
	if err := s.waitForReady(port, 10*time.Second, func() { beats++ }, nil); err != nil {
		t.Fatalf("waitForReady = %v", err)
	}
	if beats == 0 {
		t.Fatal("heartbeat was never called during the wait")
	}
}
