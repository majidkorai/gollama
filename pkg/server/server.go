package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

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
	s.mux.HandleFunc("/api/v1/models", s.handleModels)
	s.mux.HandleFunc("/api/v1/models/delete", s.handleModelDelete)
	s.mux.HandleFunc("/api/v1/models/pull", s.handleModelPull)
	s.mux.HandleFunc("/api/v1/instances", s.handleInstances)
	s.mux.HandleFunc("/api/v1/instances/stop", s.handleInstanceStop)
	s.mux.HandleFunc("/api/v1/instances/logs", s.handleInstanceLogs)
	s.mux.HandleFunc("/api/v1/chat", s.handleChat)
	s.mux.HandleFunc("/", s.handleUI)
}

func (s *Server) Start() error {
	addr := fmt.Sprintf(":%s", s.port)
	log.Printf("gollama listening on %s", addr)
	log.Printf("Web UI: http://localhost%s", addr)
	return http.ListenAndServe(addr, s.mux)
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

	idx := model.LoadIndex()
	info, ok := idx[req.Name]
	if !ok {
		jsonError(w, "model not found", 404)
		return
	}

	if err := os.Remove(info.BlobPath); err != nil && !os.IsNotExist(err) {
		jsonError(w, fmt.Sprintf("error deleting file: %v", err), 500)
		return
	}

	delete(idx, req.Name)
	model.SaveIndex(idx)

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
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResponse(w, map[string]string{"status": "ok", "model": req.Model})
}

func (s *Server) handleInstances(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		jsonResponse(w, s.mgr.List())
	case http.MethodPost:
		var req struct {
			Model string   `json:"model"`
			Port  int      `json:"port"`
			Flags []string `json:"flags"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, err.Error(), 400)
			return
		}
		inst, err := s.mgr.Start(req.Model, req.Port, req.Flags)
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

	target := fmt.Sprintf("http://127.0.0.1:%d/v1/chat/completions", port)
	resp, err := http.Post(target, "application/json", strings.NewReader(string(body)))
	if err != nil {
		jsonError(w, fmt.Sprintf("proxy error: %v", err), 502)
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var timingData struct {
		Timings *struct {
			PredictedPerSecond float64 `json:"predicted_per_second"`
		} `json:"timings"`
	}
	if json.Unmarshal(respBody, &timingData) == nil && timingData.Timings != nil {
		s.mgr.UpdateTokens(port, timingData.Timings.PredictedPerSecond)
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
	fmt.Fprint(w, ui.Page)
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
