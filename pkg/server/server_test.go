package server

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/majidkorai/gollama/pkg/manager"
	"github.com/majidkorai/gollama/pkg/model"
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

func TestInstanceLogTail(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	logDir := filepath.Join(home, ".gollama", "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatal(err)
	}
	content := "line one\n\rprogress bar 50%\r\nline two\nline three\n"
	if err := os.WriteFile(filepath.Join(logDir, "port-8080.log"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	got := instanceLogTail(8080, 2)
	want := "line two | line three"
	if got != want {
		t.Fatalf("tail = %q, want %q", got, want)
	}
	// A missing log file degrades to a hint, not a panic.
	if got := instanceLogTail(9999, 3); got != "no log available" {
		t.Fatalf("missing log = %q, want %q", got, "no log available")
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

// ── Warmup ───────────────────────────────────────────────────────────────

// setupWarmupEnv installs a throwaway HOME with a config containing one text
// profile (model "fake-1b", fixed port 8131), an index entry for the model,
// and a dummy llama-server binary that just sleeps.
func setupWarmupEnv(t *testing.T) {
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
	// Save before NewManager (it calls LoadConfig, which creates a default).
	model.SaveConfig(&model.Config{
		DefaultFlags:  []string{"--ctx-size", "512"},
		ProxyDefaults: []string{},
		Profiles: map[string]model.Profile{
			"fake": {Model: "fake-1b", Flags: []string{"--port", "8131"}, Type: "text"},
		},
		IdleTTL: 0,
	})
}

func postWarmup(t *testing.T, s *Server, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/warmup", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.mux.ServeHTTP(rec, req)
	return rec
}

func TestWarmupStartsAndIdempotent(t *testing.T) {
	setupWarmupEnv(t)
	s := New(manager.NewManager(), "8080")
	t.Cleanup(func() { s.mgr.Stop(8131) })

	rec := postWarmup(t, s, `{"profile":"fake"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("warmup = %d: %s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("bad JSON: %v", err)
	}
	if out["status"] != "starting" || out["port"] != float64(8131) || out["type"] != "text" {
		t.Fatalf("unexpected warmup response: %v", out)
	}

	// Second warmup is idempotent — same instance, no restart.
	rec = postWarmup(t, s, `{"profile":"fake"}`)
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("bad JSON: %v", err)
	}
	if out["status"] != "running" || out["port"] != float64(8131) {
		t.Fatalf("second warmup = %v, want running on 8131", out)
	}
}

func TestWarmupModelAutoDetect(t *testing.T) {
	setupWarmupEnv(t)
	s := New(manager.NewManager(), "8080")
	t.Cleanup(func() { s.mgr.Stop(8131) })

	rec := postWarmup(t, s, `{"model":"fake-1b"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("warmup by model = %d: %s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &out)
	if out["status"] != "starting" || out["profile"] != "fake" {
		t.Fatalf("auto-detect response: %v", out)
	}
}

func TestWarmupUnknowns(t *testing.T) {
	setupWarmupEnv(t)
	s := New(manager.NewManager(), "8080")

	if rec := postWarmup(t, s, `{"profile":"nope"}`); rec.Code != http.StatusNotFound {
		t.Fatalf("unknown profile = %d, want 404: %s", rec.Code, rec.Body.String())
	}
	if rec := postWarmup(t, s, `{"model":"nope-1b"}`); rec.Code != http.StatusNotFound {
		t.Fatalf("unknown model = %d, want 404: %s", rec.Code, rec.Body.String())
	}
	if rec := postWarmup(t, s, `{}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("empty body = %d, want 400: %s", rec.Code, rec.Body.String())
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/warmup", nil)
	rec := httptest.NewRecorder()
	s.mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET warmup = %d, want 405", rec.Code)
	}
}
