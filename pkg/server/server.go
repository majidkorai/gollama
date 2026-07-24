package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/majidkorai/gollama/pkg/manager"
	"github.com/majidkorai/gollama/pkg/model"
	"github.com/majidkorai/gollama/pkg/ui"
)

type Server struct {
	mgr     *manager.Manager
	port    string
	version string
	mux     *http.ServeMux
}

func New(mgr *manager.Manager, port string) *Server {
	return NewWithVersion(mgr, port, "")
}

func NewWithVersion(mgr *manager.Manager, port, version string) *Server {
	s := &Server{
		mgr:     mgr,
		port:    port,
		version: version,
		mux:     http.NewServeMux(),
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
	s.mux.HandleFunc("/api/v1/models/repo-files", s.handleRepoFiles)
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
	s.mux.HandleFunc("/api/v1/version", s.handleVersion)
	s.mux.HandleFunc("/api/v1/restart", s.handleRestart)
	s.mux.HandleFunc("/v1/models", s.handleV1Models)
	s.mux.HandleFunc("/v1/models/", s.handleV1ModelsByID)
	s.mux.HandleFunc("/v1/chat/completions", s.handleV1ChatCompletions)
	s.mux.HandleFunc("/v1/completions", s.handleV1Completions)
	s.mux.HandleFunc("/v1/images/generations", s.handleV1ImageGenerations)
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

func (s *Server) handleRepoFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", 405)
		return
	}
	repo := r.URL.Query().Get("repo")
	if repo == "" {
		jsonError(w, "repo query parameter is required", 400)
		return
	}
	files, err := model.ListRepoGGUFFiles(repo)
	if err != nil {
		jsonError(w, err.Error(), 502)
		return
	}
	if files == nil {
		files = []model.RepoGGUFFile{}
	}
	jsonResponse(w, files)
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
		// Ignore write errors — client may disconnect
		_, _ = w.Write([]byte(buf.String()))
		flusher.Flush()
	}

	// Send initial event so browser gets HTTP headers immediately
	writeSSE("", map[string]string{"status": "connecting"})

	err := model.PullModelWithCallbackContext(r.Context(), modelRef, func(pct float64, done, total int64, speed string, part, totalParts int) {
		evt := map[string]interface{}{
			"pct":   pct,
			"done":  done,
			"total": total,
			"speed": speed,
		}
		if totalParts > 1 {
			evt["part"] = part
			evt["total_parts"] = totalParts
		}
		writeSSE("progress", evt)
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
		inst, err := s.mgr.Start(req.Model, req.Port, req.Flags, req.ReplaceFlags, nil)
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

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, map[string]string{
		"version":      s.version,
		"llama_server": model.LlamaServerVersion(),
		"backend":      model.LlamaServerBackend(),
	})
}

func (s *Server) handleRestart(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", 405)
		return
	}
	// Stop all running instances before restart
	for _, inst := range s.mgr.List() {
		s.mgr.Stop(inst.Port)
	}
	jsonResponse(w, map[string]string{"status": "restarting"})
	flusher, ok := w.(http.Flusher)
	if ok {
		flusher.Flush()
	}

	// If running under systemd, use systemctl for a clean restart
	if _, err := os.Stat("/etc/systemd/system/gollama.service"); err == nil {
		go func() {
			time.Sleep(500 * time.Millisecond)
			exec.Command("systemctl", "restart", "gollama").Run()
		}()
		return
	}

	// Otherwise spawn a new process and exit
	go func() {
		time.Sleep(500 * time.Millisecond)
		cmd := exec.Command(os.Args[0], os.Args[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		if err := cmd.Start(); err != nil {
			log.Printf("restart failed: %v", err)
			return
		}
		log.Printf("restarted with PID %d", cmd.Process.Pid)
		os.Exit(0)
	}()
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
		if v, ok := incoming["proxy_defaults"]; ok {
			if arr, ok := v.([]interface{}); ok {
				flags := make([]string, 0, len(arr))
				for _, item := range arr {
					if s, ok := item.(string); ok {
						flags = append(flags, s)
					}
				}
				cfg.ProxyDefaults = flags
			}
		}
		if v, ok := incoming["profiles"]; ok {
			if m, ok := v.(map[string]interface{}); ok {
				profiles := make(map[string]model.Profile, len(m))
				for name, val := range m {
					if pMap, ok := val.(map[string]interface{}); ok {
						p := model.Profile{}
						if modelStr, ok := pMap["model"].(string); ok {
							p.Model = modelStr
						}
						if desc, ok := pMap["description"].(string); ok {
							p.Description = desc
						}
						if typ, ok := pMap["type"].(string); ok {
							p.Type = typ
						}
						if sr, ok := pMap["strip_reasoning"]; ok {
							if b, ok := sr.(bool); ok && b {
								p.StripReasoning = &b
							}
						}
						if mr, ok := pMap["merge_reasoning"]; ok {
							if b, ok := mr.(bool); ok && b {
								p.MergeReasoning = &b
							}
						}
						if envRaw, ok := pMap["env"].(map[string]interface{}); ok {
							env := make(map[string]string, len(envRaw))
							for ek, ev := range envRaw {
								if s, ok := ev.(string); ok {
									env[ek] = s
								}
							}
							p.Env = env
						}
						if flagsArr, ok := pMap["flags"].([]interface{}); ok {
							for _, item := range flagsArr {
								if s, ok := item.(string); ok {
									p.Flags = append(p.Flags, s)
								}
							}
						}
						profiles[name] = p
					}
				}
				for name := range cfg.Profiles {
					if _, ok := profiles[name]; !ok {
						delete(cfg.Profiles, name)
					}
				}
				for name, p := range profiles {
					cfg.Profiles[name] = p
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

func (s *Server) waitForInstanceReady(port int) {
	healthClient := &http.Client{Timeout: 2 * time.Second}
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := healthClient.Get(baseURL + "/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				return
			}
		}
		time.Sleep(500 * time.Millisecond)
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

	// Wait for instance to be ready (model loaded, health check passing)
	s.waitForInstanceReady(port)

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

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		for k, v := range resp.Header {
			w.Header()[k] = v
		}
		w.WriteHeader(resp.StatusCode)
		w.Write(bodyBytes)
		return
	}

	if isStream {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		flusher, ok := w.(http.Flusher)
		if !ok {
			jsonError(w, "streaming not supported", 500)
			return
		}
		sentDone := false
		hadFinishReason := false
		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadString('\n')
			if err != nil && err != io.EOF {
				sentDone = true
				break
			}
			s.mgr.TouchActivity(port)
			w.Write([]byte(line))
			flusher.Flush()
			// Parse complete SSE data line for usage tracking
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "data: ") {
				data := strings.TrimPrefix(trimmed, "data: ")
				if data == "[DONE]" {
					sentDone = true
					if !hadFinishReason {
						w.Write([]byte("data: {\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n"))
						flusher.Flush()
					}
					break
				}
				if !hadFinishReason {
					var check struct {
						Choices []struct {
							FinishReason *string `json:"finish_reason"`
						} `json:"choices"`
					}
					if json.Unmarshal([]byte(data), &check) == nil {
						for _, c := range check.Choices {
							if c.FinishReason != nil && *c.FinishReason != "" {
								hadFinishReason = true
								break
							}
						}
					}
				}
				var payload struct {
					Usage *struct {
						CompletionTokens int64 `json:"completion_tokens"`
					} `json:"usage"`
				}
				if json.Unmarshal([]byte(data), &payload) == nil && payload.Usage != nil {
					s.mgr.AddCompletionTokens(port, payload.Usage.CompletionTokens)
				}
			}
			if err == io.EOF {
				sentDone = true
				break
			}
		}
		if !sentDone && !hadFinishReason {
			w.Write([]byte("data: {\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n"))
			flusher.Flush()
		}
		if !sentDone {
			w.Write([]byte("data: [DONE]\n"))
			flusher.Flush()
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

	// Use cached index — do NOT call ListModels (which reads GGUF metadata from every file)
	idx := model.LoadIndex()
	now := time.Now().Unix()
	data := make([]openAIModel, 0, len(idx)*2)
	seen := map[string]bool{}
	for _, info := range idx {
		shortID := info.ShortName
		if shortID != "" && !seen[shortID] {
			seen[shortID] = true
			data = append(data, openAIModel{
				ID:      shortID,
				Object:  "model",
				Created: now,
				OwnedBy: "gollama",
			})
		}
		fullID := info.Name
		if fullID != "" && !seen[fullID] {
			seen[fullID] = true
			data = append(data, openAIModel{
				ID:      fullID,
				Object:  "model",
				Created: now,
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

func (s *Server) handleV1ImageGenerations(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", 405)
		return
	}

	var body json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid request body", 400)
		return
	}

	var reqMap map[string]interface{}
	json.Unmarshal(body, &reqMap)
	prompt, _ := reqMap["prompt"].(string)
	if prompt == "" {
		jsonError(w, "prompt field is required", 400)
		return
	}

	modelName, _ := reqMap["model"].(string)
	profileName, _ := reqMap["profile"].(string)

	cfg := model.LoadConfig()

	// Resolve profile: explicit or auto-detect by model name
	if profileName == "" {
		for name, p := range cfg.Profiles {
			if p.Type == "image" {
				if modelName != "" && p.Model != "" && strings.EqualFold(modelName, p.Model) {
					profileName = name
					break
				}
				if modelName == "" {
					profileName = name
					break
				}
			}
		}
	}

	// Ensure resolved profile is an image type
	if profileName != "" {
		p, ok := cfg.Profiles[profileName]
		if !ok || p.Type != "image" {
			jsonError(w, fmt.Sprintf("profile %q is not an image model", profileName), 400)
			return
		}
		if modelName == "" {
			modelName = p.Model
		}
	}
	if modelName == "" {
		jsonError(w, "no image model found. Define an image profile in settings.", 400)
		return
	}

	// Stop text instances only if they've been idle for a short grace period
	// (no active requests). This prevents interrupting in-flight SSE streams
	// while still allowing preemption when the text model is sitting idle.
	for _, inst := range s.mgr.List() {
		if inst.Status == "running" && inst.Type != "image" {
			sinceActivity := time.Since(inst.LastActivity)
			if sinceActivity < 10*time.Second && sinceActivity >= 0 {
				log.Printf("text instance %q (port %d) active %v ago — deferring image", inst.Model, inst.Port, sinceActivity.Round(time.Second))
				w.Header().Set("Retry-After", "10")
				jsonError(w, fmt.Sprintf("text model %q is busy, retry image generation later", inst.Model), 503)
				return
			}
			log.Printf("stopping idle text instance %q (port %d) for image generation (idle %v)", inst.Model, inst.Port, sinceActivity.Round(time.Second))
			s.mgr.Stop(inst.Port)
		}
	}

	inst := s.mgr.FindInstanceByModel(modelName)
	if inst == nil {
		startPort := 9081
		// Find the next free port in the 9081+ range
		for _, existing := range s.mgr.List() {
			if existing.Type == "image" && existing.Port >= startPort {
				startPort = existing.Port + 1
			}
		}

		var profileEnv map[string]string
		if profileName != "" {
			if p, ok := cfg.Profiles[profileName]; ok {
				profileEnv = p.Env
			}
		}

		inst, err := s.mgr.StartImage(modelName, startPort, profileEnv)
		if err != nil {
			jsonError(w, fmt.Sprintf("starting image model %q: %v", modelName, err), 500)
			return
		}
		log.Printf("auto-started image model %q on port %d", modelName, inst.Port)
		if profileName != "" {
			s.mgr.SetProfile(inst.Port, profileName)
		}
		go s.waitForInstanceReady(inst.Port)
		w.Header().Set("Retry-After", "5")
		jsonError(w, fmt.Sprintf("image model %q is starting, retry in a few seconds", modelName), 503)
		return
	}
	s.mgr.TouchActivity(inst.Port)
	s.waitForInstanceReady(inst.Port)

	target := fmt.Sprintf("http://127.0.0.1:%d/v1/images/generations", inst.Port)

	proxyCtx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()

	proxyReq, err := http.NewRequestWithContext(proxyCtx, "POST", target, strings.NewReader(string(body)))
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

	for k, v := range resp.Header {
		w.Header()[k] = v
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (s *Server) proxyToInstance(w http.ResponseWriter, r *http.Request, targetPath string) {
	var body json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid request body", 400)
		return
	}

	var reqMap map[string]interface{}
	json.Unmarshal(body, &reqMap)

	// Only apply schema sanitization when tools or response_format are present.
	// These are expensive recursive walks that we skip for simple chat requests.
	if _, hasTools := reqMap["tools"]; hasTools {
		stripAdditionalProperties(reqMap)
		sanitizeSchemaPatterns(reqMap)
		simplifyToolSchemas(reqMap)
	} else if _, hasRespFmt := reqMap["response_format"]; hasRespFmt {
		stripAdditionalProperties(reqMap)
		sanitizeSchemaPatterns(reqMap)
	}
	body, _ = json.Marshal(reqMap)

	modelName, _ := reqMap["model"].(string)
	if modelName == "" {
		jsonError(w, "model field is required", 400)
		return
	}

	profileName, _ := reqMap["profile"].(string)

	// Resolve flags: explicit profile, auto-detect from model name, or proxy defaults
	cfg := model.LoadConfig()
	if profileName == "" {
		// Auto-detect profile by model name
		for name, p := range cfg.Profiles {
			if p.Model != "" && strings.EqualFold(modelName, p.Model) {
				profileName = name
				break
			}
		}
	}
	var launchFlags []string
	if profileName != "" {
		launchFlags = cfg.ProfileFlags(profileName)
		log.Printf("using model profile %q for model %q", profileName, modelName)
	} else {
		launchFlags = cfg.ProxyFlags()
	}

	// Check if a different instance is already running — stop it (single-instance mode)
	if running := s.mgr.List(); len(running) > 0 {
		for _, inst := range running {
			if inst.Status == "running" && inst.Model != modelName {
				log.Printf("stopping existing instance %q (port %d) for new model %q", inst.Model, inst.Port, modelName)
				s.mgr.Stop(inst.Port)
			}
		}
	}

	inst := s.mgr.FindInstanceByModel(modelName)
	if inst == nil {
		blob, err := model.ResolveModelBlob(modelName)
		if err == nil && blob != "" {
			var profileEnv map[string]string
			if profileName != "" {
				if p, ok := cfg.Profiles[profileName]; ok {
					profileEnv = p.Env
				}
			}
			inst, err = s.mgr.Start(modelName, 0, launchFlags, false, profileEnv)
			if err != nil {
				jsonError(w, fmt.Sprintf("starting model %q: %v", modelName, err), 500)
				return
			}
			log.Printf("auto-started model %q on port %d", modelName, inst.Port)
			// Record which profile launched this instance
			if profileName != "" {
				s.mgr.SetProfile(inst.Port, profileName)
			}
			go s.waitForInstanceReady(inst.Port)
			w.Header().Set("Retry-After", "5")
			jsonError(w, fmt.Sprintf("model %q is starting, retry in a few seconds", modelName), 503)
			return
		} else {
			available := s.mgr.List()
			names := make([]string, 0, len(available))
			for _, a := range available {
				if a.Status == "running" {
					names = append(names, a.Model)
				}
			}
			if len(names) == 0 {
				jsonError(w, fmt.Sprintf("model %q not found in index and no instances running. Pull it from the gollama UI first.", modelName), 404)
			} else {
				jsonError(w, fmt.Sprintf("model %q not found in index. running instances: %s", modelName, strings.Join(names, ", ")), 404)
			}
			return
		}
	}
	s.mgr.TouchActivity(inst.Port)
	s.waitForInstanceReady(inst.Port)

	isStream, _ := reqMap["stream"].(bool)
	target := fmt.Sprintf("http://127.0.0.1:%d%s", inst.Port, targetPath)

	var proxyCtx context.Context
	proxyCtx = r.Context()
	if isStream {
		proxyCtx = context.Background()
	} else {
		var cancel context.CancelFunc
		proxyCtx, cancel = context.WithTimeout(r.Context(), 10*time.Minute)
		defer cancel()
	}

	proxyReq, err := http.NewRequestWithContext(proxyCtx, "POST", target, strings.NewReader(string(body)))
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

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		errMsg := strings.ToLower(string(bodyBytes))
		// If grammar parsing failed, retry with simplified tool schemas to avoid GBNF complexity limits
		if (strings.Contains(errMsg, "grammar") || strings.Contains(errMsg, "parse error")) && reqMap["tools"] != nil {
			log.Printf("grammar parse error, retrying with simplified tools")
			simplifyToolSchemas(reqMap)
			retryBody, _ := json.Marshal(reqMap)
			retryCtx, retryCancel := context.WithTimeout(r.Context(), 10*time.Minute)
			defer retryCancel()
			retryReq, err := http.NewRequestWithContext(retryCtx, "POST", target, strings.NewReader(string(retryBody)))
			if err == nil {
				retryReq.Header.Set("Content-Type", "application/json")
				resp2, err2 := http.DefaultClient.Do(retryReq)
				if err2 == nil && resp2.StatusCode < 400 {
					defer resp2.Body.Close()
					for k, v := range resp2.Header {
						w.Header()[k] = v
					}
					w.WriteHeader(resp2.StatusCode)
					io.Copy(w, resp2.Body)
					return
				}
				if err2 != nil {
					log.Printf("retry also failed: %v", err2)
				}
				if resp2 != nil {
					resp2.Body.Close()
				}
			}
		}
		for k, v := range resp.Header {
			w.Header()[k] = v
		}
		w.WriteHeader(resp.StatusCode)
		w.Write(bodyBytes)
		return
	}

	if isStream {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		flusher, ok := w.(http.Flusher)
		if !ok {
			jsonError(w, "streaming not supported", 500)
			return
		}
		shouldStrip := shouldStripReasoning(cfg, profileName)
		shouldMerge := shouldMergeReasoning(cfg, profileName)
		inThink := false
		thinkBuf := ""
		sentDone := false
		hadFinishReason := false
		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadString('\n')
			if err != nil && err != io.EOF {
				sentDone = true
				break
			}
			s.mgr.TouchActivity(inst.Port)
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "data: ") {
				data := strings.TrimPrefix(trimmed, "data: ")
				if data == "[DONE]" {
					sentDone = true
					if !hadFinishReason {
						w.Write([]byte("data: {\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n"))
						flusher.Flush()
					}
					w.Write([]byte(line))
					flusher.Flush()
					break
				}
			var cleaned []byte
			if shouldMerge {
				// Merge mode: keep <think> content in content field for clients that
				// don't parse reasoning_content (e.g. opencode's TUI).
				// Skip extractThinkStream since it strips <think> into reasoning_content.
				cleaned = []byte(data)
			} else if shouldStrip {
				cleaned = stripContentThinkTags([]byte(data))
				cleaned = stripReasoningContent(cleaned)
			} else {
				data = extractThinkStream(data, &inThink, &thinkBuf)
				cleaned = []byte(data)
			}
				if !hadFinishReason {
					var check struct {
						Choices []struct {
							FinishReason *string `json:"finish_reason"`
						} `json:"choices"`
					}
					if json.Unmarshal(cleaned, &check) == nil {
						for _, c := range check.Choices {
							if c.FinishReason != nil && *c.FinishReason != "" {
								hadFinishReason = true
								break
							}
						}
					}
				}
				line = "data: " + string(cleaned) + "\n"
				var usage struct {
					Usage *struct {
						CompletionTokens int64 `json:"completion_tokens"`
					} `json:"usage"`
				}
				if json.Unmarshal([]byte(data), &usage) == nil && usage.Usage != nil {
					s.mgr.AddCompletionTokens(inst.Port, usage.Usage.CompletionTokens)
				}
			}
			w.Write([]byte(line))
			flusher.Flush()
			if err == io.EOF {
				sentDone = true
				break
			}
		}
		if !sentDone && !hadFinishReason {
			w.Write([]byte("data: {\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n"))
			flusher.Flush()
		}
		if !sentDone {
			w.Write([]byte("data: [DONE]\n"))
			flusher.Flush()
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

	if shouldStripReasoning(cfg, profileName) {
		respBody = stripContentThinkTags(respBody)
		respBody = stripReasoningContent(respBody)
	} else {
		respBody = convertCompleteThink(respBody)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(respBody)
}

// shouldMergeReasoning checks whether reasoning_content should be moved into
// content for the given profile. Default is false (keep separate).
func shouldMergeReasoning(cfg *model.Config, profileName string) bool {
	if profileName == "" || cfg == nil {
		return false
	}
	p, ok := cfg.Profiles[profileName]
	if !ok {
		return false
	}
	return p.MergeReasoning != nil && *p.MergeReasoning
}

// shouldStripReasoning checks whether reasoning_content should be stripped
// for the given profile. Default is false (show reasoning).
func shouldStripReasoning(cfg *model.Config, profileName string) bool {
	if profileName == "" || cfg == nil {
		return false
	}
	p, ok := cfg.Profiles[profileName]
	if !ok {
		return false
	}
	return p.StripReasoning != nil && *p.StripReasoning
}

// mergeReasoningContent converts "reasoning_content" into "content" in real-time
// so it's visible as regular text to clients that don't parse reasoning_content.
func mergeReasoningContent(data []byte) []byte {
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return data
	}
	choices, _ := obj["choices"].([]interface{})
	if len(choices) == 0 {
		return data
	}
	choice, _ := choices[0].(map[string]interface{})
	if choice == nil {
		return data
	}
	delta, ok := choice["delta"].(map[string]interface{})
	if !ok {
		return data
	}

	if rc, ok := delta["reasoning_content"].(string); ok {
		// Send reasoning as visible content immediately instead of accumulating
		delete(delta, "reasoning_content")
		if existing, ok := delta["content"].(string); ok {
			delta["content"] = existing + rc
		} else {
			delta["content"] = rc
		}
	}

	cleaned, _ := json.Marshal(obj)
	return cleaned
}

// stripReasoningContent removes "reasoning_content" fields from JSON responses.
// <think> tags are stripped unconditionally by stripContentThinkTags before this runs.
func stripReasoningContent(data []byte) []byte {
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return data // not JSON, pass through
	}
	// Navigate to choices[0].delta (streaming) or choices[0].message (non-streaming)
	choices, _ := obj["choices"].([]interface{})
	if len(choices) == 0 {
		return data
	}
	choice, _ := choices[0].(map[string]interface{})
	if choice == nil {
		return data
	}
	for _, key := range []string{"delta", "message"} {
		if msg, ok := choice[key].(map[string]interface{}); ok {
			delete(msg, "reasoning_content")
			break
		}
	}
	cleaned, _ := json.Marshal(obj)
	return cleaned
}

// extractThinkStream handles <think>..</think> extraction from streaming SSE chunks.
// Accumulates thinking across chunks, flushes as reasoning_content on </think>.
func extractThinkStream(data string, inThink *bool, buf *string) string {
	// Parse the JSON and extract the content from delta/message
	deltaKey, content, err := parseDeltaContent([]byte(data))
	if err != nil || deltaKey == "" {
		return data
	}

	if *inThink {
		if ei := strings.Index(content, "</think>"); ei >= 0 {
			*buf += content[:ei]
			after := content[ei+8:]
			type choicesObj = map[string]interface{}
			var obj choicesObj
			json.Unmarshal([]byte(data), &obj)
			choices, _ := obj["choices"].([]interface{})
			if len(choices) > 0 {
				if ch, ok := choices[0].(choicesObj); ok {
					if d, ok := ch["delta"].(choicesObj); ok {
						d["reasoning_content"] = *buf
						delete(d, "content")
					}
				}
			}
			cleaned, _ := json.Marshal(obj)
			*buf = ""
			*inThink = false
			if after != "" {
				var obj2 choicesObj
				json.Unmarshal(cleaned, &obj2)
				chs, _ := obj2["choices"].([]interface{})
				if len(chs) > 0 {
					if ch, ok := chs[0].(choicesObj); ok {
						if d, ok := ch["delta"].(choicesObj); ok {
							d["content"] = after
						}
					}
				}
				cleaned2, _ := json.Marshal(obj2)
				return string(cleaned2)
			}
			return string(cleaned)
		}
		*buf += content
		return rewriteDeltaContent([]byte(data), deltaKey, "")
	}

	if si := strings.Index(content, "<think>"); si >= 0 {
		afterOpen := content[si+7:]
		if ei := strings.Index(afterOpen, "</think>"); ei >= 0 {
			// Complete think in one chunk — rewrite directly
			pre := content[:si]
			reasoning := afterOpen[:ei]
			post := afterOpen[ei+8:]
			result := []byte(data)
			type choicesObj = map[string]interface{}
			var obj choicesObj
			if json.Unmarshal(result, &obj) == nil {
				if chs, ok := obj["choices"].([]interface{}); ok && len(chs) > 0 {
					if ch, ok := chs[0].(choicesObj); ok {
						if d, ok := ch["delta"].(choicesObj); ok {
							d["reasoning_content"] = reasoning
							if pre != "" || post != "" {
								d["content"] = pre + post
							} else {
								delete(d, "content")
							}
						}
					}
				}
			}
			cleaned, _ := json.Marshal(obj)
			return string(cleaned)
		}
		// Start buffering
		if si > 0 {
			result := rewriteDeltaContent([]byte(data), deltaKey, content[:si])
			*buf = afterOpen
			*inThink = true
			return result
		} else {
			result := rewriteDeltaContent([]byte(data), deltaKey, "")
			*buf = afterOpen
			*inThink = true
			return result
		}
	}

	return data
}

// parseDeltaContent extracts the content string from delta or message field.
func parseDeltaContent(raw []byte) (key string, content string, err error) {
	var obj map[string]interface{}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return "", "", err
	}
	choices, _ := obj["choices"].([]interface{})
	if len(choices) == 0 {
		return "", "", nil
	}
	choice, _ := choices[0].(map[string]interface{})
	if choice == nil {
		return "", "", nil
	}
	for _, k := range []string{"delta", "message"} {
		if msg, ok := choice[k].(map[string]interface{}); ok {
			if c, ok := msg["content"].(string); ok {
				return k, c, nil
			}
		}
	}
	return "", "", nil
}

// rewriteDeltaContent sets the content field in delta/message to a new value.
func rewriteDeltaContent(raw []byte, key string, value string) string {
	if key == "" {
		return string(raw)
	}
	var obj map[string]interface{}
	if json.Unmarshal(raw, &obj) != nil {
		return string(raw)
	}
	choices, _ := obj["choices"].([]interface{})
	if len(choices) == 0 {
		return string(raw)
	}
	choice, _ := choices[0].(map[string]interface{})
	if choice == nil {
		return string(raw)
	}
	if msg, ok := choice[key].(map[string]interface{}); ok {
		if value == "" {
			delete(msg, "content")
		} else {
			msg["content"] = value
		}
	}
	cleaned, _ := json.Marshal(obj)
	return string(cleaned)
}

// convertCompleteThink extracts <think>..</think> from message content
// into reasoning_content for a complete (non-streaming) response.
func convertCompleteThink(data []byte) []byte {
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return data
	}
	choices, _ := obj["choices"].([]interface{})
	if len(choices) == 0 {
		return data
	}
	choice, _ := choices[0].(map[string]interface{})
	if choice == nil {
		return data
	}
	msg, _ := choice["message"].(map[string]interface{})
	if msg == nil {
		return data
	}
	c, _ := msg["content"].(string)
	if c == "" {
		return data
	}
	si := strings.Index(c, "<think>")
	if si < 0 {
		return data
	}
	afterOpen := c[si+7:]
	ei := strings.Index(afterOpen, "</think>")
	if ei < 0 {
		return data
	}
	reasoning := afterOpen[:ei]
	rest := afterOpen[ei+8:]
	msg["reasoning_content"] = reasoning
	if si > 0 || rest != "" {
		msg["content"] = c[:si] + rest
	} else {
		delete(msg, "content")
	}
	cleaned, _ := json.Marshal(obj)
	return cleaned
}

// stripThinkTags removes <think>...</think> sections from a string.
func stripThinkTags(s string) string {
	for {
		si := strings.Index(s, "<think>")
		if si < 0 {
			break
		}
		ei := strings.Index(s[si:], "</think>")
		if ei < 0 {
			break
		}
		s = s[:si] + s[si+ei+8:]
	}
	return s
}

// stripContentThinkTags removes <think>...</think> from content fields in JSON responses.
// This is called when strip_reasoning is enabled for a profile.
func stripContentThinkTags(data []byte) []byte {
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return data
	}
	choices, _ := obj["choices"].([]interface{})
	if len(choices) == 0 {
		return data
	}
	choice, _ := choices[0].(map[string]interface{})
	if choice == nil {
		return data
	}
	for _, key := range []string{"delta", "message"} {
		if msg, ok := choice[key].(map[string]interface{}); ok {
			if c, ok := msg["content"].(string); ok {
				msg["content"] = stripThinkTags(c)
			}
			break
		}
	}
	cleaned, _ := json.Marshal(obj)
	return cleaned
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

// simplifyToolSchemas strips nested properties/items from tool parameter schemas
// to prevent llama-server's grammar parser from exceeding repetition limits.
// Only keeps the top-level parameter names and types; nested objects become
// untyped placeholders.
func simplifyToolSchemas(v interface{}) {
	tools, ok := v.(map[string]interface{})
	if !ok {
		return
	}
	toolsArr, ok := tools["tools"].([]interface{})
	if !ok {
		return
	}
	for _, t := range toolsArr {
		tool, ok := t.(map[string]interface{})
		if !ok {
			continue
		}
		fn, ok := tool["function"].(map[string]interface{})
		if !ok {
			continue
		}
		params, ok := fn["parameters"].(map[string]interface{})
		if !ok {
			continue
		}
		props, ok := params["properties"].(map[string]interface{})
		if !ok {
			continue
		}
		for _, propVal := range props {
			propObj, ok := propVal.(map[string]interface{})
			if !ok {
				continue
			}
			// Remove nested properties/items from any object/array parameter
			delete(propObj, "properties")
			delete(propObj, "items")
		}
	}
}

// stripAdditionalProperties removes "additionalProperties" from JSON schemas
// to prevent llama-server's grammar parser from generating massive
// character-by-character matching rules that exceed repetition limits.
func stripAdditionalProperties(v interface{}) {
	switch val := v.(type) {
	case map[string]interface{}:
		delete(val, "additionalProperties")
		for _, child := range val {
			stripAdditionalProperties(child)
		}
	case []interface{}:
		for _, item := range val {
			stripAdditionalProperties(item)
		}
	}
}

// sanitizeSchemaPatterns walks a JSON-like map and sanitizes regex patterns
// in JSON schemas so llama-server's grammar parser doesn't reject them.
// Replaces unsupported regex escapes (\S, \D, \W, \s, \d, \w) with GBNF-safe equivalents.
func sanitizeSchemaPatterns(v interface{}) {
	replacer := strings.NewReplacer(
		`\S`, `[^ ]`,
		`\D`, `[^0-9]`,
		`\W`, `[^a-zA-Z0-9_]`,
		`\s`, `[ ]`,
		`\d`, `[0-9]`,
		`\w`, `[a-zA-Z0-9_]`,
	)
	switch val := v.(type) {
	case map[string]interface{}:
		for k, child := range val {
			if k == "pattern" {
				if s, ok := child.(string); ok {
					fixed := replacer.Replace(s)
					needsStart := !strings.HasPrefix(fixed, "^")
					needsEnd := !strings.HasSuffix(fixed, "$")
					if needsStart {
						fixed = "^" + fixed
					}
					if needsEnd {
						fixed = fixed + "$"
					}
					val[k] = fixed
				}
			} else {
				sanitizeSchemaPatterns(child)
			}
		}
	case []interface{}:
		for _, item := range val {
			sanitizeSchemaPatterns(item)
		}
	}
}
