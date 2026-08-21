//go:build !windows

package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/majidkorai/gollama/pkg/manager"
	"github.com/majidkorai/gollama/pkg/model"
)

// ── Warmup (unix-only: depends on HOME redirect + shell-script dummies) ──

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
	s := New(manager.NewManagerNoRecovery(), "8080")
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
	s := New(manager.NewManagerNoRecovery(), "8080")
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
	s := New(manager.NewManagerNoRecovery(), "8080")

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
