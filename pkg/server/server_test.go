package server

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/majidkorai/gollama/pkg/manager"
	"github.com/majidkorai/gollama/pkg/model"
)

// newTestServer returns a Server backed by a manager isolated under a
// temporary HOME so no real config/instances are touched. Both HOME and
// USERPROFILE are set so isolation holds on Windows too (os.UserHomeDir
// reads USERPROFILE there).
func newTestServer(t *testing.T) *Server {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	return New(manager.NewManagerNoRecovery(), "8080")
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

// ── Phase 1: security ─────────────────────────────────

func saveTestConfig(t *testing.T, token string) {
	t.Helper()
	model.SaveConfig(&model.Config{
		DefaultFlags: []string{"--ctx-size", "512"},
		Profiles:     map[string]model.Profile{},
		IdleTTL:      0,
		APIToken:     token,
	})
}

func getWithAuth(t *testing.T, s *Server, target, header, query string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	if header != "" {
		req.Header.Set("Authorization", header)
	}
	if query != "" {
		req.URL.RawQuery = query
	}
	rec := httptest.NewRecorder()
	s.mux.ServeHTTP(rec, req)
	return rec
}

func TestAuthDisabledWhenNoToken(t *testing.T) {
	s := newTestServer(t)
	saveTestConfig(t, "")
	if rec := getWithAuth(t, s, "/api/v1/version", "", ""); rec.Code != http.StatusOK {
		t.Fatalf("no token: /api/v1/version = %d, want 200 (auth disabled)", rec.Code)
	}
}

func TestAuthRequiresToken(t *testing.T) {
	s := newTestServer(t)
	saveTestConfig(t, "test-token-123")

	if rec := getWithAuth(t, s, "/api/v1/version", "", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("no creds = %d, want 401", rec.Code)
	}
	if rec := getWithAuth(t, s, "/api/v1/version", "Bearer wrong", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong token = %d, want 401", rec.Code)
	}
	if rec := getWithAuth(t, s, "/api/v1/version", "Bearer test-token-123", ""); rec.Code != http.StatusOK {
		t.Fatalf("bearer header = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if rec := getWithAuth(t, s, "/api/v1/version", "", "token=test-token-123"); rec.Code != http.StatusOK {
		t.Fatalf("query token = %d, want 200", rec.Code)
	}
	if rec := getWithAuth(t, s, "/v1/chat/completions", "", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("OpenAI route without token = %d, want 401", rec.Code)
	}
	// UI assets stay open — the UI is a viewer, its fetches carry the token.
	if rec := getWithAuth(t, s, "/", "", ""); rec.Code != http.StatusOK {
		t.Fatalf("UI route = %d, want 200 (open)", rec.Code)
	}
	if rec := getWithAuth(t, s, "/logo.svg", "", ""); rec.Code != http.StatusOK {
		t.Fatalf("logo = %d, want 200 (open)", rec.Code)
	}

	// A rotated token takes effect immediately (no restart).
	cfg := model.LoadConfig()
	cfg.APIToken = "rotated-token"
	model.SaveConfig(cfg)
	if rec := getWithAuth(t, s, "/api/v1/version", "Bearer test-token-123", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("stale token after rotation = %d, want 401", rec.Code)
	}
	if rec := getWithAuth(t, s, "/api/v1/version", "Bearer rotated-token", ""); rec.Code != http.StatusOK {
		t.Fatalf("rotated token = %d, want 200", rec.Code)
	}
}

func TestHandleConfigToken(t *testing.T) {
	s := newTestServer(t)
	saveTestConfig(t, "")

	post := func(body, token string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/config/token", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		s.mux.ServeHTTP(rec, req)
		return rec
	}

	rec := post(`{"action":"regenerate"}`, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("regenerate = %d: %s", rec.Code, rec.Body.String())
	}
	var out struct {
		Status   string `json:"status"`
		APIToken string `json:"api_token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("bad JSON: %v", err)
	}
	if out.Status != "regenerated" || len(out.APIToken) != 64 {
		t.Fatalf("regenerate response: %q token len %d", out.Status, len(out.APIToken))
	}
	if model.LoadConfig().APIToken != out.APIToken {
		t.Fatal("regenerated token not persisted to config")
	}
	if rec := getWithAuth(t, s, "/api/v1/version", "Bearer "+out.APIToken, ""); rec.Code != http.StatusOK {
		t.Fatalf("new token on protected route = %d, want 200", rec.Code)
	}

	// Clearing now requires the token that regenerate just set.
	if rec := post(`{"action":"clear"}`, out.APIToken); rec.Code != http.StatusOK {
		t.Fatalf("clear = %d: %s", rec.Code, rec.Body.String())
	}
	if tok := model.LoadConfig().APIToken; tok != "" {
		t.Fatalf("after clear, token = %q, want empty", tok)
	}
	if rec := getWithAuth(t, s, "/api/v1/version", "", ""); rec.Code != http.StatusOK {
		t.Fatalf("after clear: unauthenticated request = %d, want 200", rec.Code)
	}

	if rec := post(`{"action":"bogus"}`, ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("bogus action = %d, want 400", rec.Code)
	}
}

func TestChatIDTraversal(t *testing.T) {
	s := newTestServer(t)
	saveTestConfig(t, "")

	badIDs := []string{
		"abc/def",
		".hidden",
		"abc..def",
		"has space",
		"-leading",
		strings.Repeat("a", 65),
	}
	for _, id := range badIDs {
		for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete} {
			req := httptest.NewRequest(method, "/api/v1/chats/"+url.PathEscape(id), nil)
			rec := httptest.NewRecorder()
			s.mux.ServeHTTP(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Errorf("%s /api/v1/chats/%q = %d, want 400", method, id, rec.Code)
			}
		}
	}
	// A valid id passes validation and reaches the (missing) file → 404.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/chats/abc-123_X", nil)
	rec := httptest.NewRecorder()
	s.mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("valid chat id = %d, want 404 (validation passed)", rec.Code)
	}
}

func TestModelDeleteRejectsPathsOutsideModelsDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	modelsDir := filepath.Join(home, ".gollama", "models")
	if err := os.MkdirAll(modelsDir, 0755); err != nil {
		t.Fatal(err)
	}
	// A sibling dir that shares the old path *prefix* — the HasPrefix bug.
	evilDir := filepath.Join(home, ".gollama", "models-evil")
	if err := os.MkdirAll(evilDir, 0755); err != nil {
		t.Fatal(err)
	}
	evilFile := filepath.Join(evilDir, "x.gguf")
	if err := os.WriteFile(evilFile, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	model.SaveIndex(map[string]model.ModelInfo{
		"evil":    {Name: "evil", BlobPath: evilFile, Size: 1},
		"outside": {Name: "outside", BlobPath: "/tmp/gollama-test-outside.gguf", Size: 1},
	})

	s := New(manager.NewManagerNoRecovery(), "8080")
	for _, name := range []string{"evil", "outside"} {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/models/delete", strings.NewReader(`{"name":"`+name+`"}`))
		rec := httptest.NewRecorder()
		s.mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("delete %q = %d, want 400: %s", name, rec.Code, rec.Body.String())
		}
	}
	if _, err := os.Stat(evilFile); err != nil {
		t.Errorf("file outside models dir was deleted: %v", err)
	}
}

// TestListenAddress verifies the v3.8.0 bind-address logic: loopback by
// default, explicit override honored, empty falls back to loopback. The
// actual socket bind is stdlib http.ListenAndServe (smoke-tested on the VM);
// we avoid a real pick-a-port/rebind here because that pattern flakes on
// Windows CI.
func TestListenAddress(t *testing.T) {
	s := newTestServer(t)
	if s.listen != "127.0.0.1" {
		t.Fatalf("default listen = %q, want 127.0.0.1", s.listen)
	}
	if got := NewWithListen(manager.NewManagerNoRecovery(), "9080", "test", "0.0.0.0").listen; got != "0.0.0.0" {
		t.Fatalf("explicit listen = %q, want 0.0.0.0", got)
	}
	if got := NewWithListen(manager.NewManagerNoRecovery(), "9080", "test", "10.0.0.5").listen; got != "10.0.0.5" {
		t.Fatalf("explicit listen = %q, want 10.0.0.5", got)
	}
	if got := NewWithListen(manager.NewManagerNoRecovery(), "9080", "test", "").listen; got != "127.0.0.1" {
		t.Fatalf("empty listen = %q, want 127.0.0.1", got)
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
