package server

import (
	"bufio"
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/majidkorai/gollama/pkg/manager"
	"github.com/majidkorai/gollama/pkg/model"
	"github.com/majidkorai/gollama/pkg/ui"
)

type Server struct {
	mgr     *manager.Manager
	coord   *manager.Coordinator // serializes model switches (P3-T1)
	port    string
	listen  string // bind address; "127.0.0.1" by default (v3.8.0)
	version string
	mux     *http.ServeMux
}

func New(mgr *manager.Manager, port string) *Server {
	return NewWithListen(mgr, port, "", "127.0.0.1")
}

func NewWithVersion(mgr *manager.Manager, port, version string) *Server {
	return NewWithListen(mgr, port, version, "127.0.0.1")
}

func NewWithListen(mgr *manager.Manager, port, version, listen string) *Server {
	if listen == "" {
		listen = "127.0.0.1"
	}
	s := &Server{
		mgr:     mgr,
		coord:   manager.NewCoordinator(mgr),
		port:    port,
		listen:  listen,
		version: version,
		mux:     http.NewServeMux(),
	}
	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("/logo.svg", s.handleLogo)
	// All API + OpenAI-proxy routes are behind the shared-secret token
	// (no-op while the config has no token). UI assets stay open — the UI
	// is a viewer and attaches the token to its own fetches.
	s.mux.HandleFunc("/api/v1/models", s.requireAuth(s.handleModels))
	s.mux.HandleFunc("/api/v1/models/search", s.requireAuth(s.handleModelSearch))
	s.mux.HandleFunc("/api/v1/models/delete", s.requireAuth(s.handleModelDelete))
	s.mux.HandleFunc("/api/v1/models/pull", s.requireAuth(s.handleModelPull))
	s.mux.HandleFunc("/api/v1/models/repo-files", s.requireAuth(s.handleRepoFiles))
	s.mux.HandleFunc("/api/v1/models/pull/stream", s.requireAuth(s.handleModelPullStream))
	s.mux.HandleFunc("/api/v1/instances", s.requireAuth(s.handleInstances))
	s.mux.HandleFunc("/api/v1/instances/stop", s.requireAuth(s.handleInstanceStop))
	s.mux.HandleFunc("/api/v1/instances/logs", s.requireAuth(s.handleInstanceLogs))
	s.mux.HandleFunc("/api/v1/warmup", s.requireAuth(s.handleWarmup))
	s.mux.HandleFunc("/api/v1/config/default-flags", s.requireAuth(s.handleDefaultFlags))
	s.mux.HandleFunc("/api/v1/presets", s.requireAuth(s.handlePresets))
	s.mux.HandleFunc("/api/v1/chats", s.requireAuth(s.handleChats))
	s.mux.HandleFunc("/api/v1/chats/", s.requireAuth(s.handleChatByID))
	s.mux.HandleFunc("/api/v1/chat", s.requireAuth(s.handleChat))
	s.mux.HandleFunc("/api/v1/config", s.requireAuth(s.handleConfig))
	s.mux.HandleFunc("/api/v1/config/token", s.requireAuth(s.handleConfigToken))
	s.mux.HandleFunc("/api/v1/version", s.requireAuth(s.handleVersion))
	s.mux.HandleFunc("/api/v1/restart", s.requireAuth(s.handleRestart))
	s.mux.HandleFunc("/v1/models", s.requireAuth(s.handleV1Models))
	s.mux.HandleFunc("/v1/models/", s.requireAuth(s.handleV1ModelsByID))
	s.mux.HandleFunc("/v1/chat/completions", s.requireAuth(s.handleV1ChatCompletions))
	s.mux.HandleFunc("/v1/completions", s.requireAuth(s.handleV1Completions))
	s.mux.HandleFunc("/v1/images/generations", s.requireAuth(s.handleV1ImageGenerations))
	s.mux.HandleFunc("/api/v1/image-models", s.requireAuth(s.handleImageModels))
	s.mux.HandleFunc("/api/v1/image-models/search", s.requireAuth(s.handleImageModelSearch))
	s.mux.HandleFunc("/api/v1/image-models/install", s.requireAuth(s.handleImageModelInstall))
	s.mux.HandleFunc("/", s.handleUI)
}

// requireAuth guards API routes with the shared-secret token from the
// config. An empty token disables auth (legacy behavior) — the warning
// about that is logged once at startup. The token is read from disk per
// request, matching the codebase's LoadConfig-per-request convention, so
// a token regenerated in Settings takes effect immediately.
func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if token := model.LoadConfig().APIToken; token != "" {
			if !tokenMatches(r, token) {
				w.Header().Set("WWW-Authenticate", "Bearer")
				jsonError(w, "unauthorized: missing or invalid API token", 401)
				return
			}
		}
		next(w, r)
	}
}

// tokenMatches accepts either Authorization: Bearer <token> or ?token=<token>
// (query form for curl/cron simplicity). Comparison is constant-time.
func tokenMatches(r *http.Request, token string) bool {
	cand := ""
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		cand = strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
	}
	if cand == "" {
		cand = r.URL.Query().Get("token")
	}
	return cand != "" && subtle.ConstantTimeCompare([]byte(cand), []byte(token)) == 1
}

func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%s", s.listen, s.port)
	slog.Info("gollama listening", "addr", addr)
	if s.listen == "127.0.0.1" || s.listen == "localhost" {
		slog.Info("Web UI available (loopback only — use --listen 0.0.0.0 for LAN access)", "url", fmt.Sprintf("http://%s:%s", s.listen, s.port))
	} else {
		slog.Info("Web UI available", "url", fmt.Sprintf("http://%s:%s", s.listen, s.port))
	}
	if model.LoadConfig().APIToken == "" {
		slog.Warn("no API token, gollama is unauthenticated")
	}
	return http.ListenAndServe(addr, s.mux)
}

func (s *Server) handleLogo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Write([]byte(ui.LogoSVG))
}

func (s *Server) handleModels(w http.ResponseWriter, r *http.Request) {
	// ?refresh=1 forces a models-dir scan (otherwise it's throttled).
	if r.URL.Query().Get("refresh") == "1" {
		model.ScanModelsForce()
	}
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
			return model.ErrNotFound
		}
		blobPath = info.BlobPath
		delete(idx, req.Name)
		return nil
	}); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			jsonError(w, "model not found", 404)
		} else {
			jsonError(w, err.Error(), 500)
		}
		return
	}

	absPath, err := filepath.Abs(blobPath)
	modelsDir, err2 := filepath.Abs(model.ModelsDir())
	if err != nil || err2 != nil {
		jsonError(w, "invalid model path", 400)
		return
	}
	// filepath.Rel (not HasPrefix): a prefix check would also admit
	// sibling dirs like <modelsDir>-evil when they share a path prefix.
	rel, relErr := filepath.Rel(modelsDir, absPath)
	if relErr != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		jsonError(w, "invalid model path", 400)
		return
	}

	if err := os.Remove(absPath); err != nil && !os.IsNotExist(err) {
		jsonError(w, fmt.Sprintf("error deleting file: %v", err), 500)
		return
	}

	slog.Info("model deleted", "model", req.Name)
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
		if errors.Is(err, model.ErrAlreadyExists) {
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
		if errors.Is(err, model.ErrAlreadyExists) {
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
		// Quick launch goes through the coordinator (P3-T1): serialized
		// against proxy/warmup switches, but explicit-launch semantics —
		// never stops other instances, never reuses one.
		inst, err := s.coord.SwitchAndStart(manager.SwitchRequest{
			Model:        req.Model,
			Mode:         manager.SwitchExplicit,
			Port:         req.Port,
			Flags:        req.Flags,
			ReplaceFlags: req.ReplaceFlags,
		})
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
	// Tail only (P4-T3): logs grow unbounded and carry \r progress spam.
	data, err := manager.TailLogFile(logFile, 256*1024)
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

	// If running under systemd (system or user unit), use systemctl for a
	// clean restart
	if args := systemdRestartArgs(); args != nil {
		go func() {
			time.Sleep(500 * time.Millisecond)
			exec.Command("systemctl", args...).Run()
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
			slog.Error("restart failed", "error", err)
			return
		}
		slog.Info("restarted", "pid", cmd.Process.Pid)
		os.Exit(0)
	}()
}

// systemdRestartArgs returns the systemctl arguments for a clean gollama
// restart if a gollama unit is installed (system unit first, then user
// unit), else nil (caller falls back to re-exec).
func systemdRestartArgs() []string {
	if _, err := os.Stat("/etc/systemd/system/gollama.service"); err == nil {
		return []string{"restart", "gollama"}
	}
	if home, err := os.UserHomeDir(); err == nil {
		if p := filepath.Join(home, ".config", "systemd", "user", "gollama.service"); fileExists(p) {
			return []string{"--user", "restart", "gollama.service"}
		}
	}
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
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
		// api_token: present + non-empty sets it, present + empty disables
		// auth (the empty value persists in config.json on purpose).
		if v, ok := incoming["api_token"]; ok {
			if s, ok := v.(string); ok {
				cfg.APIToken = s
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
						if bp, ok := pMap["binary_path"].(string); ok {
							p.BinaryPath = bp
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
						// Image-specific fields
						if steps, ok := pMap["steps"].(float64); ok {
							v := int(steps)
							p.Steps = &v
						}
						if guidance, ok := pMap["guidance"].(float64); ok {
							p.Guidance = &guidance
						}
						if size, ok := pMap["size"].(string); ok {
							p.Size = &size
						}
						if n, ok := pMap["n"].(float64); ok {
							v := int(n)
							p.N = &v
						}
						profiles[name] = p
					}
				}
			// Only prune missing profiles when we actually received profiles
			// (prevents accidental wipe when saveProfiles sends an empty set)
			if len(profiles) > 0 {
				for name := range cfg.Profiles {
					if _, ok := profiles[name]; !ok {
						delete(cfg.Profiles, name)
					}
				}
			}
				for name, p := range profiles {
					cfg.Profiles[name] = p
				}
			}
		}
		if err := model.SaveConfig(cfg); err != nil {
			jsonError(w, err.Error(), 500)
			return
		}
		jsonResponse(w, map[string]string{"status": "saved"})
	default:
		http.Error(w, "method not allowed", 405)
	}
}

// handleConfigToken regenerates (POST {"action":"regenerate"}) or clears
// (POST {"action":"clear"}) the API token. Used by the Settings page.
func (s *Server) handleConfigToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	var req struct {
		Action string `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, err.Error(), 400)
		return
	}
	cfg := model.LoadConfig()
	switch req.Action {
	case "regenerate":
		token, err := model.GenerateAPIToken()
		if err != nil {
			jsonError(w, err.Error(), 500)
			return
		}
		cfg.APIToken = token
		if err := model.SaveConfig(cfg); err != nil {
			jsonError(w, err.Error(), 500)
			return
		}
		jsonResponse(w, map[string]string{"status": "regenerated", "api_token": token})
	case "clear":
		cfg.APIToken = ""
		if err := model.SaveConfig(cfg); err != nil {
			jsonError(w, err.Error(), 500)
			return
		}
		slog.Warn("API token cleared, gollama is now unauthenticated")
		jsonResponse(w, map[string]string{"status": "cleared"})
	default:
		jsonError(w, "action must be 'regenerate' or 'clear'", 400)
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
	// Chat ids become filenames — reject anything that isn't a plain id
	// (blocks ../, absolute paths, embedded separators).
	if !model.ValidChatID(id) {
		jsonError(w, "invalid chat ID", 400)
		return
	}
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

// instanceLogTail returns the last n non-empty lines of an instance's log,
// used to explain why a model failed to start.
func instanceLogTail(port int, n int) string {
	logFile := filepath.Join(model.GollamaDir(), "logs", fmt.Sprintf("port-%d.log", port))
	// Tail only (P4-T3): a small window is plenty for last-lines diagnostics.
	data, err := manager.TailLogFile(logFile, 64*1024)
	if err != nil {
		return "no log available"
	}
	lines := strings.Split(string(data), "\n")
	out := make([]string, 0, n)
	for i := len(lines) - 1; i >= 0 && len(out) < n; i-- {
		if strings.Contains(lines[i], "\r") {
			continue // progress bar line
		}
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		out = append([]string{line}, out...)
	}
	if len(out) == 0 {
		return "log is empty"
	}
	return strings.Join(out, " | ")
}

// sseErrorChunk renders an error as an OpenAI-style SSE data event so failures
// can be delivered after the stream has already started (headers sent).
func sseErrorChunk(message string) []byte {
	payload, _ := json.Marshal(map[string]interface{}{
		"error": map[string]string{
			"message": message,
			"type":    "server_error",
			"code":    "gollama_proxy_error",
		},
	})
	return append(append([]byte("data: "), payload...), '\n', '\n')
}

// proxyFail reports a proxy failure: as an SSE error event when the response
// has already started streaming, otherwise as a JSON error.
func proxyFail(w http.ResponseWriter, streamed bool, msg string) {
	if streamed {
		flusher := w.(http.Flusher)
		w.Write(sseErrorChunk(msg))
		w.Write([]byte("data: [DONE]\n\n"))
		flusher.Flush()
		return
	}
	jsonError(w, msg, 502)
}

// waitForInstanceReady polls the instance's /health endpoint until the model
// is loaded (HTTP 200) or the load timeout expires. On failure the returned
// error includes a tail of the instance log so callers can tell *why* the
// model did not come up, not just that it didn't.
func (s *Server) waitForInstanceReady(port int) error {
	return s.waitForReady(port, model.LoadTimeout(), nil, nil)
}

// waitForReady is the core of waitForInstanceReady with hooks: beat is called
// at every poll interval (used to emit SSE heartbeats during the wait), and
// abort returns true to stop waiting early (e.g. client disconnected).
func (s *Server) waitForReady(port int, timeout time.Duration, beat func(), abort func() bool) error {
	healthClient := &http.Client{Timeout: 2 * time.Second}
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if abort != nil && abort() {
			return fmt.Errorf("client disconnected while model was loading")
		}
		resp, err := healthClient.Get(baseURL + "/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		} else if status := s.mgr.InstanceStatus(port); status != "" && status != "running" {
			// The process exited before serving — fail fast instead of
			// polling a dead port for the full timeout.
			return fmt.Errorf("model process exited before becoming ready: %s", instanceLogTail(port, 5))
		}
		if beat != nil {
			beat()
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("model did not become ready within %s: %s", timeout.Round(time.Second), instanceLogTail(port, 5))
}

// sseOpts carries the per-call knobs for proxySSE.
type sseOpts struct {
	strip bool    // strip reasoning_content / think tags from the stream
	merge bool    // merge reasoning_content into content (takes precedence over strip)
	touch func()  // called per upstream line (activity touch); nil = no-op
}

// postToInstance sends a completion request to an instance, retrying the
// 503 "loading model" responses that llama.cpp can answer for a short window
// after /health flips to 200. heartbeat (if non-nil) is emitted between
// retries so streaming clients stay alive; the retry deadline is the same
// load timeout used for cold starts (model.LoadTimeout). A genuine 503
// (no "loading" in the body) is returned as-is for the caller to pass
// through.
func (s *Server) postToInstance(ctx context.Context, target string, body []byte, port int, heartbeat func()) (*http.Response, error) {
	if heartbeat == nil {
		heartbeat = func() {}
	}
	loadDeadline := time.Now().Add(model.LoadTimeout())
	for {
		proxyReq, err := http.NewRequestWithContext(ctx, http.MethodPost, target, strings.NewReader(string(body)))
		if err != nil {
			return nil, err
		}
		proxyReq.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(proxyReq)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusServiceUnavailable {
			return resp, nil
		}
		loadingBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if !strings.Contains(strings.ToLower(string(loadingBody)), "loading") {
			resp.Body = io.NopCloser(bytes.NewReader(loadingBody))
			return resp, nil // genuine 503 error — caller passes it through
		}
		if time.Now().After(loadDeadline) || ctx.Err() != nil {
			resp.Body = io.NopCloser(bytes.NewReader(loadingBody))
			return resp, nil
		}
		slog.Info("instance still loading model, retrying", "port", port)
		heartbeat()
		time.Sleep(2 * time.Second)
	}
}

// proxySSE forwards an upstream SSE response to the client with the shared
// streaming semantics used by both /v1/chat/completions and /api/v1/chat:
//   - exactly one "data: [DONE]" (forwarded from upstream, or synthesized
//     when upstream closes without one),
//   - a synthesized terminal finish_reason chunk when upstream omits one,
//   - the trailing usage chunk (include_usage) forwarded after finish_reason
//     so OpenAI clients receive their token counts,
//   - reasoning transforms (merge / strip / think-tag extraction) per opts.
func (s *Server) proxySSE(w http.ResponseWriter, r *http.Request, resp *http.Response, port int, opts sseOpts) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		jsonError(w, "streaming not supported", 500)
		return
	}
	touch := opts.touch
	if touch == nil {
		touch = func() {}
	}
	inThink := false
	thinkBuf := ""
	doneSent := false
	hadFinishReason := false
	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			break
		}
		if r.Context().Err() != nil {
			break
		}
		touch()
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "data: ") {
			data := strings.TrimPrefix(trimmed, "data: ")
			if data == "[DONE]" {
				if !hadFinishReason {
					w.Write([]byte("data: {\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n"))
					flusher.Flush()
				}
				if _, werr := w.Write([]byte(line)); werr == nil {
					flusher.Flush()
				}
				doneSent = true
				break
			}
			// After finish_reason, skip content chunks (prevents token leak),
			// except the trailing usage chunk (choices:[] + usage) that llama.cpp
			// emits when the client sets stream_options.include_usage. Forward it
			// so OpenAI clients receive their token usage, matching other providers.
			if hadFinishReason {
				var usageOnly struct {
					Usage struct {
						CompletionTokens int64 `json:"completion_tokens"`
					} `json:"usage"`
				}
				if json.Unmarshal([]byte(data), &usageOnly) == nil && usageOnly.Usage.CompletionTokens > 0 {
					s.mgr.AddCompletionTokens(port, usageOnly.Usage.CompletionTokens)
					if _, werr := w.Write([]byte("data: " + data + "\n")); werr != nil {
						doneSent = true
						break
					}
					flusher.Flush()
				}
				continue
			}
			var cleaned []byte
			if opts.merge {
				cleaned = mergeReasoningContent([]byte(data))
			} else if opts.strip {
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
				s.mgr.AddCompletionTokens(port, usage.Usage.CompletionTokens)
			}
		}
		if _, err := w.Write([]byte(line)); err != nil {
			doneSent = true
			break
		}
		flusher.Flush()
		if err == io.EOF {
			break
		}
	}
	// Guarantee exactly one [DONE] even when upstream closes without it.
	if !doneSent {
		if !hadFinishReason {
			w.Write([]byte("data: {\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n"))
		}
		w.Write([]byte("data: [DONE]\n"))
		flusher.Flush()
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
	if err := s.waitForInstanceReady(port); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}

	target := fmt.Sprintf("http://127.0.0.1:%d/v1/chat/completions", port)

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()

	resp, err := s.postToInstance(ctx, target, body, port, nil)
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
		s.proxySSE(w, r, resp, port, sseOpts{
			touch: func() { s.mgr.TouchActivity(port) },
		})
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

	// Resolve profile: explicit or auto-detect by model name. Deterministic
	// (P2-T5): with no model given, a single image profile is used, but
	// multiple ones are a 400 listing the names — no map-order lottery.
	if profileName == "" {
		var imageNames []string
		for name, p := range cfg.Profiles {
			if p.Type == "image" {
				imageNames = append(imageNames, name)
			}
		}
		sort.Strings(imageNames)
		switch {
		case modelName == "" && len(imageNames) == 1:
			profileName = imageNames[0]
		case modelName == "" && len(imageNames) > 1:
			jsonError(w, fmt.Sprintf("multiple image profiles configured (%s) — specify one with the \"profile\" field", strings.Join(imageNames, ", ")), 400)
			return
		case modelName != "":
			for _, name := range imageNames {
				if p := cfg.Profiles[name]; p.Model != "" && strings.EqualFold(modelName, p.Model) {
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

	// Merge profile defaults into request body for any unset fields
	if profileName != "" {
		if p, ok := cfg.Profiles[profileName]; ok && p.Type == "image" {
			if p.Steps != nil {
				if _, exists := reqMap["steps"]; !exists {
					reqMap["steps"] = *p.Steps
				}
			}
			if p.Guidance != nil {
				if _, exists := reqMap["guidance"]; !exists {
					reqMap["guidance"] = *p.Guidance
				}
			}
			if p.Size != nil {
				if _, exists := reqMap["size"]; !exists {
					reqMap["size"] = *p.Size
				}
			}
			if p.N != nil {
				if _, exists := reqMap["n"]; !exists {
					reqMap["n"] = *p.N
				}
			}
			// Re-encode the merged body for proxying
			if merged, err := json.Marshal(reqMap); err == nil {
				body = merged
			}
		}
	}

	// Switch to the image model through the coordinator (P3-T1): serialized
	// against text switches; defers with 503 + Retry-After: 30 while a text
	// model has been active in the last 30s (avoids killing an in-flight
	// agent call that still needs the GPU); stops idle text to free VRAM.
	alreadyRunning := s.mgr.FindInstanceByModel(modelName) != nil
	var profileEnv map[string]string
	if profileName != "" {
		if p, ok := cfg.Profiles[profileName]; ok {
			profileEnv = p.Env
		}
	}
	inst, err := s.coord.SwitchAndStart(manager.SwitchRequest{
		Model:   modelName,
		Mode:    manager.SwitchImage,
		Env:     profileEnv,
		Profile: profileName,
	})
	if err != nil {
		if errors.Is(err, manager.ErrBusy) {
			slog.Info("deferring image generation", "error", err)
			w.Header().Set("Retry-After", strconv.Itoa(manager.BusyRetryAfter))
			jsonError(w, err.Error(), 503)
			return
		}
		jsonError(w, fmt.Sprintf("starting image model %q: %v", modelName, err), 500)
		return
	}
	s.mgr.TouchActivity(inst.Port)
	if !alreadyRunning {
		// Just spawned: keep the 503 + Retry-After: 5 contract (clients
		// retry; readiness is polled in the background).
		slog.Info("auto-started image model", "model", modelName, "port", inst.Port)
		go s.waitForInstanceReady(inst.Port)
		w.Header().Set("Retry-After", "5")
		jsonError(w, fmt.Sprintf("image model %q is starting, retry in a few seconds", modelName), 503)
		return
	}
	if err := s.waitForInstanceReady(inst.Port); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}

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

// resolveProfile finds the model profile for a request: the explicit profile
// name if given, otherwise auto-detection by model name.
func resolveProfile(cfg *model.Config, profileName, modelName string) string {
	if profileName != "" {
		return profileName
	}
	if modelName == "" {
		return ""
	}
	for name, p := range cfg.Profiles {
		if p.Model != "" && strings.EqualFold(modelName, p.Model) {
			return name
		}
	}
	return ""
}

// handleWarmup starts a model (text or image profile) in the background so
// agents can pre-warm a model before they need it. Idempotent: a model that
// is already running is returned as-is. The request returns as soon as the
// process has been spawned — poll /api/v1/instances for the ready flag, or
// just fire the real request (it blocks until the model is up).
func (s *Server) handleWarmup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	var req struct {
		Profile string `json:"profile"`
		Model   string `json:"model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, err.Error(), 400)
		return
	}

	cfg := model.LoadConfig()
	profileName := req.Profile
	modelName := req.Model

	// Explicit profile: fill in its model. No profile given: auto-detect.
	if profileName != "" {
		p, ok := cfg.Profiles[profileName]
		if !ok {
			jsonError(w, fmt.Sprintf("profile %q not found", profileName), 404)
			return
		}
		if modelName == "" {
			modelName = p.Model
		}
	} else {
		profileName = resolveProfile(cfg, "", modelName)
		if profileName != "" && modelName == "" {
			modelName = cfg.Profiles[profileName].Model
		}
	}
	if modelName == "" {
		jsonError(w, "model or profile is required", 400)
		return
	}

	profile := cfg.Profiles[profileName] // zero value if none

	// Image profiles: same switching rules as /v1/images/generations —
	// only preempt text that has been idle for 30s.
	if profile.Type == "image" {
		// Idempotent first: an image model that is already running is
		// returned as-is, even while text is busy (warmup is pre-warming,
		// not generation).
		if inst := s.mgr.FindInstanceByModel(modelName); inst != nil {
			jsonResponse(w, map[string]interface{}{
				"status": "running", "port": inst.Port, "model": inst.Model,
				"profile": profileName, "type": "image", "ready": inst.Ready,
			})
			return
		}
		inst, err := s.coord.SwitchAndStart(manager.SwitchRequest{
			Model:   modelName,
			Mode:    manager.SwitchImage,
			Env:     profile.Env,
			Profile: profileName,
		})
		if err != nil {
			if errors.Is(err, manager.ErrBusy) {
				slog.Info("warmup: deferring image warmup", "model", modelName, "error", err)
				w.Header().Set("Retry-After", strconv.Itoa(manager.BusyRetryAfter))
				jsonError(w, err.Error(), 503)
				return
			}
			jsonError(w, fmt.Sprintf("starting image model %q: %v", modelName, err), 500)
			return
		}
		go s.waitForInstanceReady(inst.Port)
		slog.Info("warmup: image model starting", "model", modelName, "port", inst.Port)
		jsonResponse(w, map[string]interface{}{
			"status": "starting", "port": inst.Port, "model": inst.Model,
			"profile": profileName, "type": "image",
		})
		return
	}

	// Text: idempotent if an instance for this model is already running.
	if inst := s.mgr.FindInstanceByModel(modelName); inst != nil {
		jsonResponse(w, map[string]interface{}{
			"status": "running", "port": inst.Port, "model": inst.Model,
			"profile": inst.Profile, "type": "text", "ready": inst.Ready,
		})
		return
	}

	blob, err := model.ResolveModelBlob(modelName)
	if err != nil || blob == "" {
		names := make([]string, 0)
		for _, a := range s.mgr.List() {
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

	launchFlags := cfg.ProxyFlags()
	if profileName != "" {
		launchFlags = cfg.ProfileFlags(profileName)
		slog.Info("warmup: using model profile", "profile", profileName, "model", modelName)
	}
	inst, err := s.coord.SwitchAndStart(manager.SwitchRequest{
		Model:      modelName,
		Mode:       manager.SwitchText,
		Flags:      launchFlags,
		Env:        profile.Env,
		BinaryPath: profile.BinaryPath,
		Profile:    profileName,
	})
	if err != nil {
		jsonError(w, fmt.Sprintf("starting model %q: %v", modelName, err), 500)
		return
	}
	slog.Info("warmup: model starting", "model", modelName, "port", inst.Port)
	jsonResponse(w, map[string]interface{}{
		"status": "starting", "port": inst.Port, "model": inst.Model,
		"profile": profileName, "type": "text",
	})
}

// ── Image Model Management ──────────────────────────

func (s *Server) handleImageModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", 405)
		return
	}
	cfg := model.LoadConfig()
	type imgModelEntry struct {
		Name    string `json:"name"`
		ModelID string `json:"model_id"`
		Cached  bool   `json:"cached"`
		Size    int64  `json:"size"`
		Desc    string `json:"description"`
	}
	var entries []imgModelEntry
	for name, p := range cfg.Profiles {
		if p.Type == "image" && p.Model != "" {
			entry := imgModelEntry{
				Name:    name,
				ModelID: p.Model,
				Desc:    p.Description,
				Cached:  model.IsImageModelCached(p.Model),
			}
			if entry.Cached {
				entry.Size = model.ImageModelCacheSize(p.Model)
			}
			entries = append(entries, entry)
		}
	}
	if entries == nil {
		entries = []imgModelEntry{}
	}
	jsonResponse(w, entries)
}

func (s *Server) handleImageModelSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", 405)
		return
	}
	query := r.URL.Query().Get("q")
	if query == "" {
		query = "text-to-image"
	}
	results, err := model.SearchImageModels(query)
	if err != nil {
		jsonError(w, err.Error(), 502)
		return
	}
	if results == nil {
		results = []model.ImageModelSearchResult{}
	}
	jsonResponse(w, results)
}

func (s *Server) handleImageModelInstall(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", 405)
		return
	}
	var req struct {
		Name    string `json:"name"`
		ModelID string `json:"model_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, err.Error(), 400)
		return
	}
	if req.Name == "" || req.ModelID == "" {
		jsonError(w, "name and model_id are required", 400)
		return
	}

	cfg := model.LoadConfig()
	if _, exists := cfg.Profiles[req.Name]; exists {
		jsonError(w, fmt.Sprintf("profile %q already exists", req.Name), 409)
		return
	}

	// Check disk space for model dir
	home, _ := os.UserHomeDir()
	cacheDir := filepath.Join(home, ".cache", "huggingface")
	free, err := model.FreeDiskBytes(cacheDir)
	if err == nil && free < 10<<30 { // warn if less than 10GB
		slog.Warn("low disk space for image model", "model", req.ModelID, "free", model.FormatSize(int64(free)))
	}

	// Set sensible defaults based on model name
	steps := 28
	guidance := 3.5
	modelLower := strings.ToLower(req.ModelID)
	if strings.Contains(modelLower, "schnell") || strings.Contains(modelLower, "turbo") || strings.Contains(modelLower, "lcm") {
		steps = 4
	}
	if strings.Contains(modelLower, "schnell") {
		guidance = 0
	}
	if strings.Contains(modelLower, "xl") || strings.Contains(modelLower, "playground") {
		guidance = 2.5
	}

	cfg.Profiles[req.Name] = model.Profile{
		Model:       req.ModelID,
		Type:        "image",
		Flags:       []string{},
		Description: req.ModelID,
		Steps:       &steps,
		Guidance:    &guidance,
		Size:        strPtr("1024x1024"),
		N:           intPtr(1),
	}
	if err := model.SaveConfig(cfg); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}

	slog.Info("image profile installed", "name", req.Name, "model", req.ModelID)
	jsonResponse(w, map[string]string{"status": "installed", "name": req.Name, "model_id": req.ModelID})
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
	profileName = resolveProfile(cfg, profileName, modelName)
	var launchFlags []string
	if profileName != "" {
		launchFlags = cfg.ProfileFlags(profileName)
		slog.Info("using model profile", "profile", profileName, "model", modelName)
	} else {
		launchFlags = cfg.ProxyFlags()
	}

	var profileEnv map[string]string
	var profileBinary string
	if profileName != "" {
		if p, ok := cfg.Profiles[profileName]; ok {
			profileEnv = p.Env
			profileBinary = p.BinaryPath
		}
	}

	// Reject unknown models before switching: don't evict a running model
	// for a request that cannot be served.
	if s.mgr.FindInstanceByModel(modelName) == nil {
		if blob, err := model.ResolveModelBlob(modelName); err != nil || blob == "" {
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

	isStream, _ := reqMap["stream"].(bool)

	// For streaming requests, begin the SSE response immediately so the
	// client receives bytes (comment heartbeats) while a cold model loads.
	// Heartbeats keep idle/read timers alive on the client and any
	// intermediate proxies, so a multi-minute model load doesn't look like a
	// dead connection. SSE comments are ignored by OpenAI-compatible clients.
	var streamFlusher http.Flusher
	heartbeat := func() {}
	if isStream {
		if f, ok := w.(http.Flusher); ok {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")
			w.WriteHeader(http.StatusOK)
			streamFlusher = f
			w.Write([]byte(": model loading...\n\n"))
			f.Flush()
			lastBeat := time.Now()
			heartbeat = func() {
				if time.Since(lastBeat) < 10*time.Second {
					return
				}
				lastBeat = time.Now()
				w.Write([]byte(": still loading, model starting...\n\n"))
				f.Flush()
			}
		}
	}
	// Model switch (evict other models, start, wait for readiness) goes
	// through the coordinator (P3-T1): concurrent requests for the same
	// model coalesce; different models queue instead of thrashing the GPU.
	inst, err := s.coord.SwitchAndStart(manager.SwitchRequest{
		Model:       modelName,
		Mode:        manager.SwitchText,
		Flags:       launchFlags,
		Env:         profileEnv,
		BinaryPath:  profileBinary,
		Profile:     profileName,
		WaitReady:   true,
		Heartbeat:   heartbeat,
		ShouldAbort: func() bool { return r.Context().Err() != nil },
	})
	if err != nil {
		if streamFlusher != nil {
			w.Write(sseErrorChunk(err.Error()))
			w.Write([]byte("data: [DONE]\n\n"))
			streamFlusher.Flush()
			return
		}
		jsonError(w, err.Error(), 500)
		return
	}
	s.mgr.TouchActivity(inst.Port)
	target := fmt.Sprintf("http://127.0.0.1:%d%s", inst.Port, targetPath)

	var proxyCtx context.Context
	var cancel context.CancelFunc
	if isStream {
		proxyCtx, cancel = context.WithCancel(r.Context())
	} else {
		proxyCtx, cancel = context.WithTimeout(r.Context(), 10*time.Minute)
	}
	defer cancel()

	// llama.cpp can still answer 503 "loading model" for a short window
	// after /health flips to 200. postToInstance keeps retrying (with
	// heartbeats for streaming clients) until the model actually accepts
	// the request.
	resp, err := s.postToInstance(proxyCtx, target, body, inst.Port, heartbeat)
	if err != nil {
		proxyFail(w, streamFlusher != nil, fmt.Sprintf("proxy error: %v", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if streamFlusher != nil {
			// SSE 200 headers were already flushed during the model-load wait,
			// so the upstream error can't be returned as a JSON status —
			// deliver it as an SSE error event instead.
			proxyFail(w, streamFlusher != nil, fmt.Sprintf("upstream error %d: %s", resp.StatusCode, string(bodyBytes)))
			return
		}
		errMsg := strings.ToLower(string(bodyBytes))
		// If grammar parsing failed, retry with simplified tool schemas to avoid GBNF complexity limits
		if (strings.Contains(errMsg, "grammar") || strings.Contains(errMsg, "parse error")) && reqMap["tools"] != nil {
			slog.Warn("grammar parse error, retrying with simplified tools")
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
					slog.Error("retry also failed", "error", err2)
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
		// Headers were already written when streaming was started above
		// (SSE + heartbeats during model load).
		flusher := streamFlusher
		if flusher == nil {
			jsonError(w, "streaming not supported", 500)
			return
		}
		s.proxySSE(w, r, resp, inst.Port, sseOpts{
			strip: shouldStripReasoning(cfg, profileName),
			merge: shouldMergeReasoning(cfg, profileName),
			touch: func() { s.mgr.TouchActivity(inst.Port) },
		})
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

	if shouldMergeReasoning(cfg, profileName) {
		// merge_reasoning: reasoning becomes visible content. Convert think
		// tags to reasoning_content first, then fold that into content —
		// mirrors the stream path, where merge takes precedence over strip.
		respBody = convertCompleteThink(respBody)
		respBody = mergeReasoningContent(respBody)
	} else if shouldStripReasoning(cfg, profileName) {
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
// Handles both the streaming shape (choices[0].delta) and the non-streaming
// shape (choices[0].message). Pinned behavior: reasoning is appended AFTER
// any existing content in the same chunk.
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
	for _, key := range []string{"delta", "message"} {
		delta, ok := choice[key].(map[string]interface{})
		if !ok {
			continue
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
		break
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

func strPtr(s string) *string { return &s }
func intPtr(i int) *int { return &i }

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
