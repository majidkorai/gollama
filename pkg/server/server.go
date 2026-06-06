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
	s.mux.HandleFunc("/api/v1/models/delete", s.handleModelDelete)
	s.mux.HandleFunc("/api/v1/models/pull", s.handleModelPull)
	s.mux.HandleFunc("/api/v1/instances", s.handleInstances)
	s.mux.HandleFunc("/api/v1/instances/stop", s.handleInstanceStop)
	s.mux.HandleFunc("/api/v1/instances/logs", s.handleInstanceLogs)
	s.mux.HandleFunc("/api/v1/config/default-flags", s.handleDefaultFlags)
	s.mux.HandleFunc("/api/v1/presets", s.handlePresets)
	s.mux.HandleFunc("/api/v1/chats", s.handleChats)
	s.mux.HandleFunc("/api/v1/chats/", s.handleChatByID)
	s.mux.HandleFunc("/api/v1/chat", s.handleChat)
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
	if _, ok := w.(http.Flusher); !ok {
		if err := model.PullModel(req.Model); err != nil {
			jsonError(w, err.Error(), 500)
			return
		}
		jsonResponse(w, map[string]string{"status": "ok", "model": req.Model})
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(200)
	w.(http.Flusher).Flush()
	pipeR, pipeW := io.Pipe()
	defer pipeR.Close()
	go func() {
		defer pipeW.Close()
		model.PullModelWithCallback(req.Model, func(pct float64, done, total int64, speed string) {
			fmt.Fprintf(pipeW, "data: {\"pct\":%.1f,\"done\":%d,\"total\":%d,\"speed\":\"%s\"}\n\n", pct, done, total, speed)
		})
		fmt.Fprintf(pipeW, "data: {\"status\":\"done\"}\n\n")
	}()
	buf := make([]byte, 4096)
	for {
		n, err := pipeR.Read(buf)
		if n > 0 {
			w.Write(buf[:n])
			w.(http.Flusher).Flush()
		}
		if err != nil {
			break
		}
	}
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

func jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
