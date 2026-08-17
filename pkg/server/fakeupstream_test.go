package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// defaultStreamScript is the SSE payload sequence a fake upstream sends for a
// successful streaming completion: two content deltas, a finish_reason
// chunk, and the trailing usage chunk llama.cpp emits when the client sets
// stream_options.include_usage. The [DONE] marker is appended separately.
var defaultStreamScript = []string{
	`{"choices":[{"index":0,"delta":{"role":"assistant","content":"Hel"}}]}`,
	`{"choices":[{"index":0,"delta":{"content":"lo"}}]}`,
	`{"choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
	`{"choices":[],"usage":{"prompt_tokens":10,"completion_tokens":7,"total_tokens":17}}`,
}

// fakeUpstream simulates a llama-server instance well enough for the OpenAI
// proxy to talk to it: a /health endpoint plus /v1/chat/completions with
// configurable startup failures (503 "loading model", 500 grammar errors)
// and streaming or non-streaming success responses.
type fakeUpstream struct {
	mu sync.Mutex

	// loading503s is the number of completion requests that receive a 503
	// "loading model" response before succeeding.
	loading503s int
	// grammar500s is the number of requests (after the 503s are consumed)
	// that receive a 500 grammar-error response.
	grammar500s int

	// streamScript replaces defaultStreamScript when non-empty.
	streamScript []string
	// noDone suppresses the trailing [DONE] marker.
	noDone bool

	completions int
	bodies      []map[string]interface{}
}

func (f *fakeUpstream) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ok"}`)
	})
	mux.HandleFunc("/v1/chat/completions", f.handleCompletions)
	return mux
}

func (f *fakeUpstream) handleCompletions(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}

	f.mu.Lock()
	f.completions++
	n := f.completions
	f.mu.Unlock()

	if n <= f.loading503s {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprint(w, `{"error":{"message":"loading model"}}`)
		return
	}
	if n <= f.loading503s+f.grammar500s {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"error":{"message":"grammar parse error: repetition limit exceeded"}}`)
		return
	}

	f.mu.Lock()
	f.bodies = append(f.bodies, req)
	f.mu.Unlock()

	if stream, _ := req["stream"].(bool); stream {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)
		script := f.streamScript
		if len(script) == 0 {
			script = defaultStreamScript
		}
		for _, payload := range script {
			fmt.Fprintf(w, "data: %s\n\n", payload)
			flusher.Flush()
		}
		if !f.noDone {
			fmt.Fprint(w, "data: [DONE]\n\n")
			flusher.Flush()
		}
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":     "cmpl-fake",
		"object": "chat.completion",
		"choices": []map[string]interface{}{{
			"index":         0,
			"message":       map[string]interface{}{"role": "assistant", "content": "Hello!"},
			"finish_reason": "stop",
		}},
		"usage":   map[string]interface{}{"prompt_tokens": 10, "completion_tokens": 42, "total_tokens": 52},
		"timings": map[string]interface{}{"predicted_per_second": 21.5},
	})
}

// requestCount returns how many completion requests the fake has seen
// (including failed ones).
func (f *fakeUpstream) requestCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.completions
}

// lastBody returns the JSON body of the most recent successful completion.
func (f *fakeUpstream) lastBody() map[string]interface{} {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.bodies) == 0 {
		return nil
	}
	return f.bodies[len(f.bodies)-1]
}
