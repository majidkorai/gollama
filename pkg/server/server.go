package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/majidkorai/gollama/pkg/manager"
	"github.com/majidkorai/gollama/pkg/model"
	"github.com/majidkorai/gollama/pkg/ui"
)

type Server struct {
	mgr  *manager.Manager
	port string
	mux  *http.ServeMux
}

func New(mgr *manager.Manager, port string) *Server {
	s := &Server{
		mgr:  mgr,
		port: port,
		mux:  http.NewServeMux(),
	}
	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("/logo.svg", s.handleLogo)
	s.mux.HandleFunc("/api/v1/models", s.handleModels)
	s.mux.HandleFunc("/api/v1/models/search", s.handleModelSearch)
	s.mux.HandleFunc("/api/v1/models/delete", s.handleModelDelete)
	s.mux.HandleFunc("/api/v1/models/pull", s.handleModelPull)
	s.mux.HandleFunc("/api/v1/models/pull/stream", s.handleModelPullStream)
	s.mux.HandleFunc("/api/v1/instances", s.handleInstances)
	s.mux.HandleFunc("/api/v1/instances/stop", s.handleInstanceStop)
	s.mux.HandleFunc("/api/v1/instances/logs", s.handleInstanceLogs)
	s.mux.HandleFunc("/api/v1/config/default-flags", s.handleDefaultFlags)
	s.mux.HandleFunc("/api/v1/presets", s.handlePresets)
	s.mux.HandleFunc("/api/v1/chats", s.handleChats)
	s.mux.HandleFunc("/api/v1/chats/", s.handleChatByID)
	s.mux.HandleFunc("/api/v1/chat", s.handleChat)
	s.mux.HandleFunc("/api/v1/config", s.handleConfig)
	s.mux.HandleFunc("/v1/models", s.handleV1Models)
	s.mux.HandleFunc("/v1/models/", s.handleV1ModelsByID)
	s.mux.HandleFunc("/v1/chat/completions", s.handleV1ChatCompletions)
	s.mux.HandleFunc("/v1/completions", s.handleV1Completions)
	s.mux.HandleFunc("/", s.handleUI)
}

func (s *Server) Start() error {
	addr := fmt.Sprintf(":%s", s.port)
	log.Printf("gollama listening on %s", addr)
	log.Printf("Web UI: http://localhost%s", addr)
	return http.ListenAndServe(addr, s.mux)
}

func (s *Server) handleLogo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Write([]byte(ui.LogoSVG))
}

func (s *Server) handleModels(w http.ResponseWriter, r *http.Request) {
	models, err := model.ListModels()
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	if models == nil {
		models = []model.ModelInfo{}
	}
	jsonResponse(w, models)
}

func (s *Server) handleModelSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", 405)
		return
	}
	query := r.URL.Query().Get("q")
	if query == "" {
		jsonError(w, "query parameter 'q' is required", 400)
		return
	}
	results, err := model.SearchModels(query)
	if err != nil {
		jsonError(w, err.Error(), 502)
		return
	}
	if results == nil {
		results = []model.SearchResult{}
	}
	jsonResponse(w, results)
}

func (s *Server) handleModelDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", 405)
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, err.Error(), 400)
		return
	}
	if req.Name == "" {
		jsonError(w, "model name is required", 400)
		return
	}

	var blobPath string
	if err := model.UpdateIndex(func(idx map[string]model.ModelInfo) error {
		info, ok := idx[req.Name]
		if !ok {
			return fmt.Errorf("model not found")
		}
		blobPath = info.BlobPath
		delete(idx, req.Name)
		return nil
	}); err != nil {
		jsonError(w, "model not found", 404)
		return
	}

	absPath, _ := filepath.Abs(blobPath)
	modelsDir, _ := filepath.Abs(model.ModelsDir())
	if !strings.HasPrefix(absPath, modelsDir) {
		jsonError(w, "invalid model path", 400)
		return
	}

	if err := os.Remove(absPath); err != nil && !os.IsNotExist(err) {
		jsonError(w, fmt.Sprintf("error deleting file: %v", err), 500)
		return
	}

	log.Printf("model deleted: %s", req.Name)
	jsonResponse(w, map[string]string{"status": "deleted", "model": req.Name})
}

func (s *Server) handleModelPull(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Model string `json:"model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, err.Error(), 400)
		return
	}
	if err := model.PullModel(req.Model); err != nil {
		if err.Error() == "already_exists" {
			jsonResponse(w, map[string]string{"status": "exists", "model": req.Model})
		} else {
			jsonError(w, err.Error(), 500)
		}
		return
	}
	jsonResponse(w, map[string]string{"status": "ok", "model": req.Model})
}

func (s *Server) handleModelPullStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", 405)
		return
	}
	modelRef := r.URL.Query().Get("model")
	if modelRef == "" {
		jsonError(w, "model query parameter is required", 400)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		jsonError(w, "streaming not supported", 500)
		return
	}

	writeSSE := func(event string, data interface{}) {
		var buf strings.Builder
		if event != "" {
			buf.WriteString("event: ")
			buf.WriteString(event)
			buf.WriteString("\n")
		}
		buf.WriteString("data: ")
		json.NewEncoder(&buf).Encode(data)
		buf.WriteString("\n\n")
		w.Write([]byte(buf.String()))
		flusher.Flush()
	}

	// Send initial event so browser gets HTTP headers immediately
	writeSSE("", map[string]string{"status": "connecting"})

	err := model.PullModelWithCallback(modelRef, func(pct float64, done, total int64, speed string) {
		writeSSE("progress", map[string]interface{}{
			"pct":   pct,
			"done":  done,
			"total": total,
			"speed": speed,
		})
	})

	if err != nil {
		if err.Error() == "already_exists" {
			writeSSE("", map[string]string{"status": "exists"})
		} else {
			writeSSE("", map[string]string{"status": "error", "error": err.Error()})
		}
		return
	}
	writeSSE("", map[string]string{"status": "done"})
}

type flushWriter struct{ w http.ResponseWriter }

func (fw *flushWriter) Write(p []byte) (int, error) {
	n, err := fw.w.Write(p)
	if flusher, ok := fw.w.(http.Flusher); ok {
		flusher.Flush()
	}
	return n, err
}

func (s *Server) handleInstances(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		jsonResponse(w, s.mgr.List())
	case http.MethodPost:
		var req struct {
			Model        string   `json:"model"`
			Port         int      `json:"port"`
			Flags        []string `json:"flags"`
			ReplaceFlags bool     `json:"replace_flags"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, err.Error(), 400)
			return
		}
		inst, err := s.mgr.Start(req.Model, req.Port, req.Flags, req.ReplaceFlags)
		if err != nil {
			jsonError(w, err.Error(), 500)
			return
		}
		jsonResponse(w, inst)
	default:
		http.Error(w, "method not allowed", 405)
	}
}

func (s *Server) handleInstanceStop(w http.ResponseWriter, r *http.Request) {
	portStr := r.URL.Query().Get("port")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		jsonError(w, "port query parameter is required", 400)
		return
	}
	if err := s.mgr.Stop(port); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResponse(w, map[string]string{"status": "stopped"})
}

func (s *Server) handleInstanceLogs(w http.ResponseWriter, r *http.Request) {
	portStr := r.URL.Query().Get("port")
	logDir := filepath.Join(model.GollamaDir(), "logs")
	logFile := filepath.Join(logDir, fmt.Sprintf("port-%s.log", portStr))
	data, err := os.ReadFile(logFile)
	if err != nil {
		jsonError(w, "log not found", 404)
		return
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) > 100 {
		lines = lines[len(lines)-100:]
	}
	jsonResponse(w, map[string]interface{}{
		"port":  portStr,
		"lines": lines,
	})
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		jsonResponse(w, model.LoadConfig())
	case http.MethodPost:
		var incoming map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&incoming); err != nil {
			jsonError(w, err.Error(), 400)
			return
		}
		cfg := model.LoadConfig()
		if v, ok := incoming["idle_ttl"]; ok {
			if n, ok := v.(float64); ok {
				cfg.IdleTTL = int(n)
			}
		}
		if v, ok := incoming["default_flags"]; ok {
			if arr, ok := v.([]interface{}); ok {
				flags := make([]string, 0, len(arr))
				for _, item := range arr {
					if s, ok := item.(string); ok {
						flags = append(flags, s)
					}
				}
				if len(flags) > 0 {
					cfg.DefaultFlags = flags
				}
			}
		}
		model.SaveConfig(cfg)
		jsonResponse(w, map[string]string{"status": "saved"})
	default:
		http.Error(w, "method not allowed", 405)
	}
}

func (s *Server) handleDefaultFlags(w http.ResponseWriter, r *http.Request) {
	cfg := model.DefaultConfig()
	flags := cfg.DefaultFlags
	if gpuAvailable, gpuLayers := model.DetectGPU(); gpuAvailable {
		hasGPUFlag := false
		for _, f := range flags {
			if f == "--n-gpu-layers" {
				hasGPUFlag = true
				break
			}
		}
		if !hasGPUFlag {
			gpuFlags := []string{"--n-gpu-layers", strconv.Itoa(gpuLayers)}
			flags = append(gpuFlags, flags...)
		}
	}
	jsonResponse(w, flags)
}

func (s *Server) handlePresets(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		jsonResponse(w, model.GetPresets().List())
	case http.MethodPost:
		var req struct {
			Name  string   `json:"name"`
			Flags []string `json:"flags"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, err.Error(), 400)
			return
		}
		if req.Name == "" {
			jsonError(w, "name is required", 400)
			return
		}
		if err := model.GetPresets().Save(req.Name, req.Flags); err != nil {
			jsonError(w, err.Error(), 500)
			return
		}
		jsonResponse(w, map[string]string{"status": "saved"})
	case http.MethodDelete:
		name := r.URL.Query().Get("name")
		if name == "" {
			jsonError(w, "name query parameter is required", 400)
			return
		}
		model.GetPresets().Delete(name)
		jsonResponse(w, map[string]string{"status": "deleted"})
	default:
		http.Error(w, "method not allowed", 405)
	}
}

func (s *Server) handleChats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", 405)
		return
	}
	chats, err := model.ListChats()
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	if chats == nil {
		chats = []model.ChatSummary{}
	}
	jsonResponse(w, chats)
}

func (s *Server) handleChatByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/chats/")
	switch r.Method {
	case http.MethodGet:
		if id == "" {
			http.Error(w, "chat ID is required", 400)
			return
		}
		session, err := model.LoadChat(id)
		if err != nil {
			jsonError(w, "chat not found", 404)
			return
		}
		jsonResponse(w, session)
	case http.MethodPut:
		var session model.ChatSession
		if err := json.NewDecoder(r.Body).Decode(&session); err != nil {
			jsonError(w, err.Error(), 400)
			return
		}
		if session.ID == "" {
			session.ID = id
		}
		if err := model.SaveChat(&session); err != nil {
			jsonError(w, err.Error(), 500)
			return
		}
		jsonResponse(w, map[string]string{"id": session.ID})
	case http.MethodDelete:
		if err := model.DeleteChat(id); err != nil {
			jsonError(w, err.Error(), 500)
			return
		}
		jsonResponse(w, map[string]string{"status": "deleted"})
	default:
		http.Error(w, "method not allowed", 405)
	}
}

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", 405)
		return
	}
	portStr := r.URL.Query().Get("port")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		jsonError(w, "port query parameter is required", 400)
		return
	}

	var body json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid request body", 400)
		return
	}

	var reqMap map[string]interface{}
	json.Unmarshal(body, &reqMap)
	isStream, _ := reqMap["stream"].(bool)

	s.mgr.TouchActivity(port)

	target := fmt.Sprintf("http://127.0.0.1:%d/v1/chat/completions", port)

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", target, strings.NewReader(string(body)))
	if err != nil {
		jsonError(w, fmt.Sprintf("proxy error: %v", err), 502)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		jsonError(w, fmt.Sprintf("proxy error: %v", err), 502)
		return
	}
	defer resp.Body.Close()

	if isStream {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		flusher, ok := w.(http.Flusher)
		if !ok {
			jsonError(w, "streaming not supported", 500)
			return
		}
		buf := make([]byte, 256)
		for {
			n, err := resp.Body.Read(buf)
			if n > 0 {
				w.Write(buf[:n])
				flusher.Flush()
				// Parse SSE for usage data (final data event before [DONE])
				var usage struct {
					Usage *struct {
						CompletionTokens int64 `json:"completion_tokens"`
					} `json:"usage"`
				}
				if json.Unmarshal(buf[:n], &usage) == nil && usage.Usage != nil {
					s.mgr.AddCompletionTokens(port, usage.Usage.CompletionTokens)
				}
			}
			if err != nil {
				break
			}
		}
		return
	}

	respBody, _ := io.ReadAll(resp.Body)
	var responseData struct {
		Usage *struct {
			CompletionTokens int64 `json:"completion_tokens"`
		} `json:"usage"`
		Timings *struct {
			PredictedPerSecond float64 `json:"predicted_per_second"`
		} `json:"timings"`
	}
	if json.Unmarshal(respBody, &responseData) == nil {
		if responseData.Timings != nil {
			s.mgr.UpdateTokens(port, responseData.Timings.PredictedPerSecond)
		}
		if responseData.Usage != nil {
			s.mgr.AddCompletionTokens(port, responseData.Usage.CompletionTokens)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(respBody)
}

func (s *Server) handleUI(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Write([]byte(ui.Page))
}

// ── OpenAI-compatible API ─────────────────────────────────

type openAIModel struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

func (s *Server) handleV1Models(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", 405)
		return
	}
	instances := s.mgr.List()
	data := make([]openAIModel, 0, len(instances))
	for _, inst := range instances {
		if inst.Status == "running" {
			data = append(data, openAIModel{
				ID:      inst.Model,
				Object:  "model",
				Created: inst.StartedAt.Unix(),
				OwnedBy: "gollama",
			})
		}
	}
	jsonResponse(w, map[string]interface{}{
		"object": "list",
		"data":   data,
	})
}

func (s *Server) handleV1ModelsByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", 405)
		return
	}
	modelName := strings.TrimPrefix(r.URL.Path, "/v1/models/")
	if modelName == "" {
		jsonError(w, "model name is required", 400)
		return
	}
	inst := s.mgr.FindInstanceByModel(modelName)
	if inst == nil {
		jsonError(w, fmt.Sprintf("model %q not found — run 'gollama serve' and start it from the UI", modelName), 404)
		return
	}
	jsonResponse(w, openAIModel{
		ID:      inst.Model,
		Object:  "model",
		Created: inst.StartedAt.Unix(),
		OwnedBy: "gollama",
	})
}

func (s *Server) handleV1ChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", 405)
		return
	}
	s.proxyToInstance(w, r, "/v1/chat/completions")
}

func (s *Server) handleV1Completions(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", 405)
		return
	}
	s.proxyToInstance(w, r, "/v1/completions")
}

func (s *Server) proxyToInstance(w http.ResponseWriter, r *http.Request, targetPath string) {
	var body json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid request body", 400)
		return
	}

	var reqMap map[string]interface{}
	json.Unmarshal(body, &reqMap)

	modelName, _ := reqMap["model"].(string)
	if modelName == "" {
		jsonError(w, "model field is required", 400)
		return
	}

	inst := s.mgr.FindInstanceByModel(modelName)
	if inst == nil {
		// List available models to help the user
		available := s.mgr.List()
		names := make([]string, 0, len(available))
		for _, a := range available {
			if a.Status == "running" {
				names = append(names, a.Model)
			}
		}
		if len(names) == 0 {
			jsonError(w, fmt.Sprintf("model %q not found — no instances running. Start one from the gollama UI.", modelName), 404)
		} else {
			jsonError(w, fmt.Sprintf("model %q not found. running instances: %s", modelName, strings.Join(names, ", ")), 404)
		}
		return
	}
	s.mgr.TouchActivity(inst.Port)

	isStream, _ := reqMap["stream"].(bool)
	target := fmt.Sprintf("http://127.0.0.1:%d%s", inst.Port, targetPath)

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()

	proxyReq, err := http.NewRequestWithContext(ctx, "POST", target, strings.NewReader(string(body)))
	if err != nil {
		jsonError(w, fmt.Sprintf("proxy error: %v", err), 502)
		return
	}
	proxyReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(proxyReq)
	if err != nil {
		jsonError(w, fmt.Sprintf("proxy error: %v", err), 502)
		return
	}
	defer resp.Body.Close()

	if isStream {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		flusher, ok := w.(http.Flusher)
		if !ok {
			jsonError(w, "streaming not supported", 500)
			return
		}
		buf := make([]byte, 256)
		for {
			n, err := resp.Body.Read(buf)
			if n > 0 {
				w.Write(buf[:n])
				flusher.Flush()
				var usage struct {
					Usage *struct {
						CompletionTokens int64 `json:"completion_tokens"`
					} `json:"usage"`
				}
				if json.Unmarshal(buf[:n], &usage) == nil && usage.Usage != nil {
					s.mgr.AddCompletionTokens(inst.Port, usage.Usage.CompletionTokens)
				}
			}
			if err != nil {
				break
			}
		}
		return
	}

	respBody, _ := io.ReadAll(resp.Body)
	var responseData struct {
		Usage *struct {
			CompletionTokens int64 `json:"completion_tokens"`
		} `json:"usage"`
		Timings *struct {
			PredictedPerSecond float64 `json:"predicted_per_second"`
		} `json:"timings"`
	}
	if json.Unmarshal(respBody, &responseData) == nil {
		if responseData.Timings != nil {
			s.mgr.UpdateTokens(inst.Port, responseData.Timings.PredictedPerSecond)
		}
		if responseData.Usage != nil {
			s.mgr.AddCompletionTokens(inst.Port, responseData.Usage.CompletionTokens)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(respBody)
}

func jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
