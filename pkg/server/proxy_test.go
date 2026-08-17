//go:build !windows

package server

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/majidkorai/gollama/pkg/manager"
	"github.com/majidkorai/gollama/pkg/model"
)

// startProxyFixture wires up a full proxy test: an isolated HOME with a dummy
// llama-server binary (a shell script that just sleeps) and an index entry
// for "fake-1b", a manager with a running (dummy) instance on a known port,
// and the fake upstream served on that port. The dummy binary never binds
// the port, so the proxy's health check and completion requests hit the fake
// instead of a real model.
func startProxyFixture(t *testing.T, upstream *fakeUpstream) (*Server, int) {
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
	model.SaveConfig(&model.Config{
		DefaultFlags: []string{"--ctx-size", "512"},
		Profiles:     map[string]model.Profile{},
		IdleTTL:      0,
	})

	// Reserve a port, release it, and let the manager claim it.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	s := New(manager.NewManager(), "8080")
	inst, err := s.mgr.Start("fake-1b", 0, []string{"--port", strconv.Itoa(port)}, false, nil)
	if err != nil {
		t.Fatalf("starting dummy instance: %v", err)
	}
	if inst.Port != port {
		t.Fatalf("instance port = %d, want %d", inst.Port, port)
	}
	t.Cleanup(func() { s.mgr.Stop(port) })

	// Serve the fake upstream on the instance's port. Retry briefly in case
	// the port was stolen between close and rebind.
	var serveLn net.Listener
	for i := 0; i < 50; i++ {
		serveLn, err = net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if serveLn == nil {
		t.Fatalf("could not bind fake upstream to port %d: %v", port, err)
	}
	go http.Serve(serveLn, upstream.handler())
	t.Cleanup(func() { serveLn.Close() })

	return s, port
}

// postCompletions posts a body to the OpenAI chat-completions proxy and
// returns the recorded response.
func postCompletions(t *testing.T, s *Server, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.mux.ServeHTTP(rec, req)
	return rec
}

func instanceOnPort(t *testing.T, s *Server, port int) *manager.Instance {
	t.Helper()
	for _, inst := range s.mgr.List() {
		if inst.Port == port {
			return inst
		}
	}
	t.Fatalf("no instance on port %d", port)
	return nil
}

func TestProxyNonStream(t *testing.T) {
	up := &fakeUpstream{}
	s, port := startProxyFixture(t, up)

	rec := postCompletions(t, s, `{"model":"fake-1b","messages":[{"role":"user","content":"hi"}]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response is not JSON: %v", err)
	}
	choices, _ := resp["choices"].([]interface{})
	if len(choices) != 1 {
		t.Fatalf("choices = %v", resp["choices"])
	}
	msg, _ := choices[0].(map[string]interface{})["message"].(map[string]interface{})
	if msg == nil || msg["content"] != "Hello!" {
		t.Fatalf("message = %v, want content Hello!", choices[0])
	}

	inst := instanceOnPort(t, s, port)
	if inst.TotalTokens != 42 {
		t.Errorf("TotalTokens = %d, want 42 (from upstream usage)", inst.TotalTokens)
	}
	if inst.TokensPerSec != 21.5 {
		t.Errorf("TokensPerSec = %v, want 21.5 (from upstream timings)", inst.TokensPerSec)
	}
}

func TestProxyStream(t *testing.T) {
	up := &fakeUpstream{}
	s, port := startProxyFixture(t, up)

	rec := postCompletions(t, s, `{"model":"fake-1b","stream":true,"messages":[{"role":"user","content":"hi"}]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("Content-Type = %q, want text/event-stream", ct)
	}
	out := rec.Body.String()
	for _, want := range []string{
		": model loading...",   // SSE comment sent before the wait starts
		"Hel", "lo",            // content deltas
		`"completion_tokens":7`, // trailing usage chunk forwarded (regression: 2026-08-15)
		"data: [DONE]",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("stream missing %q\n--- stream ---\n%s", want, out)
		}
	}
	if n := strings.Count(out, "[DONE]"); n != 1 {
		t.Errorf("[DONE] count = %d, want exactly 1", n)
	}
	inst := instanceOnPort(t, s, port)
	if inst.TotalTokens != 7 {
		t.Errorf("TotalTokens = %d, want 7 (from trailing usage chunk)", inst.TotalTokens)
	}
}

// TestProxyStreamRetriesWhileLoading pins the cold-start path: SSE 200
// headers go out immediately, the 503 "loading model" responses are retried,
// and the client finally receives the stream.
func TestProxyStreamRetriesWhileLoading(t *testing.T) {
	up := &fakeUpstream{loading503s: 1}
	s, port := startProxyFixture(t, up)

	rec := postCompletions(t, s, `{"model":"fake-1b","stream":true,"messages":[{"role":"user","content":"hi"}]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 after 503 retry: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("Content-Type = %q, want text/event-stream", ct)
	}
	out := rec.Body.String()
	if !strings.Contains(out, "Hel") || !strings.Contains(out, "lo") {
		t.Errorf("stream missing content:\n%s", out)
	}
	if n := strings.Count(out, "[DONE]"); n != 1 {
		t.Errorf("[DONE] count = %d, want exactly 1", n)
	}
	if n := up.requestCount(); n != 2 {
		t.Errorf("upstream saw %d requests, want 2 (one 503 + one success)", n)
	}
	_ = instanceOnPort(t, s, port)
}

// TestProxyNonStreamRetriesWhileLoading is the non-streaming twin of the
// above: the proxy holds the request through the 503 window and returns the
// final 200 body.
func TestProxyNonStreamRetriesWhileLoading(t *testing.T) {
	up := &fakeUpstream{loading503s: 2}
	s, _ := startProxyFixture(t, up)

	rec := postCompletions(t, s, `{"model":"fake-1b","messages":[{"role":"user","content":"hi"}]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 after 503 retries: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Hello!") {
		t.Fatalf("response missing content: %s", rec.Body.String())
	}
	if n := up.requestCount(); n != 3 {
		t.Errorf("upstream saw %d requests, want 3 (two 503s + one success)", n)
	}
}

// TestProxyGrammarRetry pins the GBNF fallback: when the upstream rejects a
// tools request with a grammar error, the proxy retries with simplified
// schemas and returns success.
func TestProxyGrammarRetry(t *testing.T) {
	up := &fakeUpstream{grammar500s: 1}
	s, _ := startProxyFixture(t, up)

	body := `{"model":"fake-1b","messages":[{"role":"user","content":"hi"}],"tools":[{"type":"function","function":{"name":"get_weather","parameters":{"type":"object","properties":{"city":{"type":"string","properties":{"nested":{"type":"string"}}}},"required":["city"]}}}]}`
	rec := postCompletions(t, s, body)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 after grammar retry: %s", rec.Code, rec.Body.String())
	}
	if n := up.requestCount(); n != 2 {
		t.Errorf("upstream saw %d requests, want 2 (one 500 + one retry)", n)
	}
}

// TestProxyUpstreamErrorPassthrough: a 500 that is not a grammar error (or
// carries no tools) is passed through to the client unchanged.
func TestProxyUpstreamErrorPassthrough(t *testing.T) {
	up := &fakeUpstream{grammar500s: 999} // every request "fails"
	s, _ := startProxyFixture(t, up)

	rec := postCompletions(t, s, `{"model":"fake-1b","messages":[{"role":"user","content":"hi"}]}`)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 passthrough: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "grammar parse error") {
		t.Fatalf("upstream error body not passed through: %s", rec.Body.String())
	}
}

// TestHandleChatStream covers the second SSE loop (/api/v1/chat, used by the
// Web UI). It documents the current divergence from the OpenAI proxy: this
// loop does NOT forward the [DONE] marker (sentDone is set on [DONE] before
// the marker is written, and the post-loop write is guarded by !sentDone).
// P5-T1 merges the two loops; until then this test pins the behavior.
func TestHandleChatStream(t *testing.T) {
	up := &fakeUpstream{}
	s, port := startProxyFixture(t, up)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/chat?port=%d", port),
		strings.NewReader(`{"model":"default","stream":true,"messages":[{"role":"user","content":"hi"}]}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("Content-Type = %q, want text/event-stream", ct)
	}
	out := rec.Body.String()
	for _, want := range []string{"Hel", "lo"} {
		if !strings.Contains(out, want) {
			t.Errorf("stream missing %q\n--- stream ---\n%s", want, out)
		}
	}
	if n := strings.Count(out, "[DONE]"); n != 0 {
		t.Errorf("[DONE] count = %d, want 0 (documents divergence from the OpenAI proxy)", n)
	}
	inst := instanceOnPort(t, s, port)
	if inst.TotalTokens != 7 {
		t.Errorf("TotalTokens = %d, want 7 (usage chunk parsed but not forwarded)", inst.TotalTokens)
	}
}
