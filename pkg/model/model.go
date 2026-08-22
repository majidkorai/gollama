package model

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Sentinel errors so handlers can distinguish failure modes with errors.Is
// instead of string-matching (P3-T4).
var (
	ErrAlreadyExists = errors.New("model already exists")
	ErrNotFound      = errors.New("model not found")
)

type ModelInfo struct {
	Name          string `json:"name"`
	ShortName     string `json:"short_name"` // clean API-friendly name e.g. "gemma-4-12b"
	BlobPath      string `json:"blob_path"`
	Size          int64  `json:"size"`
	Architecture  string `json:"architecture,omitempty"`
	Quantization  string `json:"quantization,omitempty"`
	ContextLength uint64 `json:"context_length,omitempty"`
	BlockCount    uint32 `json:"block_count,omitempty"`
	Source        string `json:"source,omitempty"`
}

type Preset struct {
	Name  string   `json:"name"`
	Model string   `json:"model"`
	Port  int      `json:"port"`
	Flags []string `json:"flags"`
}

type Profile struct {
	Model          string            `json:"model,omitempty"`
	BinaryPath     string            `json:"binary_path,omitempty"`
	Flags          []string          `json:"flags"`
	Description    string            `json:"description,omitempty"`
	StripReasoning *bool             `json:"strip_reasoning,omitempty"`
	MergeReasoning *bool             `json:"merge_reasoning,omitempty"`
	Env            map[string]string `json:"env,omitempty"`
	Type           string            `json:"type,omitempty"` // "text" (default) or "image"

	// Image-specific defaults (optional — user can override in UI)
	Steps    *int     `json:"steps,omitempty"`
	Guidance *float64 `json:"guidance,omitempty"`
	Size     *string  `json:"size,omitempty"`
	N        *int     `json:"n,omitempty"`
}

type Config struct {
	DefaultFlags  []string          `json:"default_flags"`
	ProxyDefaults []string          `json:"proxy_defaults"` // flags for auto-launched instances; falls back to default_flags
	Profiles      map[string]Profile `json:"profiles,omitempty"`
	IdleTTL       int               `json:"idle_ttl"` // minutes; 0 = disabled
	// APIToken guards all /api/v1/* and /v1/* routes when non-empty
	// (Authorization: Bearer <token> or ?token=<token>). No omitempty on
	// purpose: an explicitly cleared token ("") must persist as disabled.
	APIToken string `json:"api_token"`
}

func ConfigFile() string {
	return filepath.Join(GollamaDir(), "config.json")
}

// IsStandaloneFlag reports whether name is a boolean (no-value) llama-server
// flag. See isStandaloneFlag (flags.go) for the derivation rules.
func IsStandaloneFlag(name string) bool {
	return isStandaloneFlag(name)
}

var standaloneFlags = map[string]bool{
	"--agent": true,
	"--backend-sampling": true,
	"--cache-idle-slots": true, "--cache-prompt": true, "--cont-batching": true, "--context-shift": true, "--cpu-moe": true,
	"--check-tensors": true,
	"--direct-io": true,
	"--embedding": true, "--escape": true,
	"--ignore-eos": true,
	"--jinja": true,
	"--kv-unified": true,
	"--list-devices": true, "--log-prefix": true, "--no-log-prefix": true, "--log-timestamps": true, "--no-log-timestamps": true, "--log-verbose": true,
	"--metrics": true, "--mlock": true, "--mmproj-auto": true, "--no-mmproj-auto": true,
	"--no-agent": true, "--no-cache-idle-slots": true, "--no-cache-prompt": true, "--no-cont-batching": true, "--no-context-shift": true, "--no-direct-io": true, "--no-escape": true, "--no-flash-attn": true, "--no-host": true, "--no-jinja": true, "--no-kv-offload": true, "--no-kv-unified": true, "--no-mmap": true, "--no-mmproj": true, "--no-mmproj-offload": true, "--no-op-offload": true, "--no-perf": true, "--no-prefill-assistant": true, "--no-repack": true, "--no-spec-draft-backend-sampling": true, "--no-ui": true, "--no-ui-mcp-proxy": true, "--no-warmup": true,
	"--offline": true, "--op-offload": true,
	"--perf": true, "--prefill-assistant": true, "--props": true,
	"--repack": true, "--rerank": true, "--reuse-port": true,
	"--spec-draft-backend-sampling": true, "--spec-draft-cpu-moe": true, "--special": true, "--spm-infill": true, "--swa-full": true,
	"--ui": true, "--ui-mcp-proxy": true,
	"--verbose": true,
}

func DefaultConfig() *Config {
	flags := []string{"--host", "127.0.0.1", "--ctx-size", "2048", "--flash-attn", "on", "--temp", "0.7"}
	return &Config{DefaultFlags: flags, IdleTTL: 30, Profiles: make(map[string]Profile)}
}

// sanitizeFlags drops orphaned values (leading values, values after a
// standalone flag, consecutive values). It is a thin wrapper over the typed
// flag model (P5-T3); ParseFlags applies the same rules.
func sanitizeFlags(flags []string) []string {
	return ParseFlags(flags).Args()
}

func LoadConfig() *Config {
	EnsureDir(GollamaDir())
	data, err := os.ReadFile(ConfigFile())
	if err != nil {
		cfg := DefaultConfig()
		if saveErr := SaveConfig(cfg); saveErr != nil {
			slog.Warn("could not save initial config", "error", saveErr)
		}
		return cfg
	}
	var cfg Config
	if json.Unmarshal(data, &cfg) != nil || cfg.DefaultFlags == nil {
		cfg = *DefaultConfig()
		if saveErr := SaveConfig(&cfg); saveErr != nil {
			slog.Warn("could not save config", "error", saveErr)
		}
	}
	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]Profile)
	}
	cfg.DefaultFlags = sanitizeFlags(cfg.DefaultFlags)
	cfg.ProxyDefaults = sanitizeFlags(cfg.ProxyDefaults)
	for name, p := range cfg.Profiles {
		p.Flags = sanitizeFlags(p.Flags)
		cfg.Profiles[name] = p
	}
	return &cfg
}

func (c *Config) ProxyFlags() []string {
	if len(c.ProxyDefaults) > 0 {
		return c.ProxyDefaults
	}
	return c.DefaultFlags
}

// ProfileFlags returns the merged flags: proxy_defaults base overridden by
// profile flags, via the typed flag model (P5-T3). Override wins for valued
// flags; standalone flags set/unset their counterpart.
func (c *Config) ProfileFlags(name string) []string {
	p, ok := c.Profiles[name]
	if !ok {
		return c.ProxyFlags()
	}
	base := ParseFlags(c.ProxyFlags())
	profile := ParseFlags(p.Flags)
	return base.Merge(profile).Args()
}

// SaveConfig writes the config atomically (tmp + rename) and returns any
// error. Callers must handle it — never write config silently (P4-T4).
func SaveConfig(cfg *Config) error {
	cfg.DefaultFlags = sanitizeFlags(cfg.DefaultFlags)
	cfg.ProxyDefaults = sanitizeFlags(cfg.ProxyDefaults)
	for name, p := range cfg.Profiles {
		p.Flags = sanitizeFlags(p.Flags)
		cfg.Profiles[name] = p
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	path := ConfigFile()
	// Backup existing config before overwriting
	if _, err := os.Stat(path); err == nil {
		if input, err := os.ReadFile(path); err == nil {
			os.WriteFile(path+".bak", input, 0644)
		}
	}
	// Atomic write: tmp + rename to prevent corruption on crash
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("writing config tmp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("renaming config into place: %w", err)
	}
	return nil
}

// GenerateAPIToken returns a fresh 32-byte random token as 64 hex chars.
func GenerateAPIToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generating API token: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

// tokenMarkerFile records that a token has been generated on this install.
// It lets EnsureAPIToken tell "never had a token" (generate one) from
// "user cleared the token" (auth disabled — respect it), regardless of
// which code path created config.json first.
func tokenMarkerFile() string {
	return filepath.Join(GollamaDir(), "api-token-generated")
}

// EnsureAPIToken returns the configured API token. If this install never
// had a token (fresh install, or a pre-v3.8 config without api_token), a
// token is generated, saved, and (token, true) is returned so the caller
// can print it once. A cleared token (marker present, token empty) is
// respected: auth stays disabled and ("", false) is returned.
func EnsureAPIToken() (string, bool) {
	cfg := LoadConfig()
	if cfg.APIToken != "" {
		return cfg.APIToken, false
	}
	if _, err := os.Stat(tokenMarkerFile()); err == nil {
		// A token was generated here before and then cleared — the user
		// disabled auth on purpose.
		return "", false
	}
	token, err := GenerateAPIToken()
	if err != nil {
		return "", false
	}
	cfg.APIToken = token
	if saveErr := SaveConfig(cfg); saveErr != nil {
		slog.Warn("could not persist API token", "error", saveErr)
	}
	if err := os.WriteFile(tokenMarkerFile(), []byte(time.Now().UTC().Format(time.RFC3339)), 0600); err != nil {
		slog.Warn("could not write token marker", "error", err)
	}
	return token, true
}

func GollamaDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".gollama")
}

func ModelsDir() string {
	return filepath.Join(GollamaDir(), "models")
}

func BinDir() string {
	return filepath.Join(GollamaDir(), "bin")
}

// LoadTimeout bounds how long gollama waits for a model to become ready
// (health check / proxy readiness). It is the single source of truth for
// both the manager's ready-poll goroutines and the server's request-holding
// deadline. Override with GOLLAMA_MODEL_LOAD_TIMEOUT (seconds); default 5m.
func LoadTimeout() time.Duration {
	const def = 5 * time.Minute
	if v := os.Getenv("GOLLAMA_MODEL_LOAD_TIMEOUT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return time.Duration(n) * time.Second
		}
	}
	return def
}

func IndexFile() string {
	return filepath.Join(GollamaDir(), "index.json")
}

func VersionFile() string {
	return filepath.Join(GollamaDir(), "llama-server-version.txt")
}

func BackendFile() string {
	return filepath.Join(GollamaDir(), "llama-server-backend.txt")
}

func LlamaServerVersion() string {
	data, err := os.ReadFile(VersionFile())
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func LlamaServerBackend() string {
	data, err := os.ReadFile(BackendFile())
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func EnsureDir(dir string) {
	os.MkdirAll(dir, 0755)
}

func DetectGPU() (available bool, layers int) {
	switch runtime.GOOS {
	case "linux":
		if gpus, _ := filepath.Glob("/proc/driver/nvidia/gpus/*"); len(gpus) > 0 {
			return true, 99
		}
		if _, err := os.Stat("/proc/driver/amdgpu/version"); err == nil {
			return true, 99
		}
	case "darwin":
		return true, 99
	case "windows":
		cmd := exec.Command("where", "nvidia-smi.exe")
		if err := cmd.Run(); err == nil {
			return true, 99
		}
	}
	return false, 0
}

type ProgressFn func(pct float64, done, total int64, speed string, part, totalParts int)

type ProgressReader struct {
	Reader     io.Reader
	Total      int64
	Done       int64
	Start      time.Time
	Name       string
	Output     io.Writer
	ProgressFn ProgressFn
	Part       int
	TotalParts int
	// TTY reports whether the progress output is a terminal. When false, the
	// \r-based progress bar is suppressed (P4-T5) so logs and systemd output
	// stay clean.
	TTY bool
}

// isTerminal reports whether f is a terminal/TTY (stdlib only).
func isTerminal(f *os.File) bool {
	if f == nil {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// progressOutputTTY reports whether the given progress output writer is a
// terminal, so \r progress rendering is appropriate.
func progressOutputTTY(progress io.Writer) bool {
	var f *os.File
	switch w := progress.(type) {
	case nil:
		f = os.Stderr
	case *os.File:
		f = w
	default:
		return false
	}
	return isTerminal(f)
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	pr.Done += int64(n)
	elapsed := time.Since(pr.Start).Seconds()
	var speed string
	if elapsed > 0 {
		rate := float64(pr.Done) / (1024 * 1024) / elapsed
		speed = fmt.Sprintf("%.1f MB/s", rate)
	}
	label := pr.Name
	if pr.TotalParts > 1 {
		label = fmt.Sprintf("%s [%d/%d]", pr.Name, pr.Part, pr.TotalParts)
	}
	if pr.ProgressFn != nil {
		var pct float64
		if pr.Total > 0 {
			pct = float64(pr.Done) * 100 / float64(pr.Total)
		}
		pr.ProgressFn(pct, pr.Done, pr.Total, speed, pr.Part, pr.TotalParts)
	} else if pr.TTY {
		// Interactive: render an in-place progress bar with \r.
		out := pr.Output
		if out == nil {
			out = os.Stderr
		}
		if pr.Total > 0 {
			pct := float64(pr.Done) * 100 / float64(pr.Total)
			fmt.Fprintf(out, "\r  %s  %.1f%%  (%s / %s)  %s    ",
				label, pct, FormatSize(pr.Done), FormatSize(pr.Total), speed)
		} else {
			fmt.Fprintf(out, "\r  %s  %s  %s       ",
				label, FormatSize(pr.Done), speed)
		}
		if err == io.EOF {
			fmt.Fprintln(pr.output())
		}
	}
	// Non-TTY with no ProgressFn: no inline progress — the \r spam would
	// corrupt logs and systemd output; pull prints start/finish to stdout.
	return n, err
}

func (pr *ProgressReader) output() io.Writer {
	if pr.Output != nil {
		return pr.Output
	}
	return os.Stderr
}

func FormatSize(bytes int64) string {
	switch {
	case bytes >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(1<<30))
	case bytes >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(1<<20))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

var (
	indexMu sync.RWMutex
)

// Model-dir scan throttling (P4-T2). The models directory is rescanned at
// most once per scanInterval so the frequently-polled GET /api/v1/models
// stays O(index) instead of O(files × GGUF headers). scanInterval <= 0
// disables throttling (tests set this).
var (
	scanMu       sync.Mutex
	lastScanTime time.Time
	scanInterval = 60 * time.Second
)

// HTTPClient is a shared HTTP client with a custom User-Agent.
// No timeout: it carries long model downloads.
var HTTPClient = &http.Client{
	Transport: &userAgentTransport{
		next: http.DefaultTransport,
	},
}

// APIClient is for short JSON API calls (HF search, GitHub releases).
// It must never hang a UI request — bounded to 30s.
var APIClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &userAgentTransport{
		next: http.DefaultTransport,
	},
}

// hfBaseURL is the HuggingFace API/CDN base. Tests override it to point at a local server.
var hfBaseURL = "https://huggingface.co"

// diskSpaceFn returns free disk bytes for a path. It is a seam so tests can
// bypass the real filesystem (whose free space varies by runner/OS) instead of
// tripping the production disk-space guard. Defaults to freeDiskBytes.
var diskSpaceFn = freeDiskBytes

type userAgentTransport struct {
	next http.RoundTripper
}

func (t *userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", "gollama/0.1.0 (+https://github.com/majidkorai/gollama)")
	return t.next.RoundTrip(req)
}

func unsafeLoadIndex() map[string]ModelInfo {
	EnsureDir(GollamaDir())
	data, err := os.ReadFile(IndexFile())
	if err != nil {
		return make(map[string]ModelInfo)
	}
	var idx map[string]ModelInfo
	if json.Unmarshal(data, &idx) != nil {
		return make(map[string]ModelInfo)
	}
	return idx
}

// unsafeSaveIndex writes the index atomically (tmp + rename) and returns any
// error. Caller must hold indexMu (P4-T4).
func unsafeSaveIndex(idx map[string]ModelInfo) error {
	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling index: %w", err)
	}
	tmp := IndexFile() + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("writing index tmp: %w", err)
	}
	if err := os.Rename(tmp, IndexFile()); err != nil {
		return fmt.Errorf("renaming index into place: %w", err)
	}
	return nil
}

func LoadIndex() map[string]ModelInfo {
	indexMu.RLock()
	defer indexMu.RUnlock()
	return unsafeLoadIndex()
}

// SaveIndex writes the index and returns any error (P4-T4).
func SaveIndex(idx map[string]ModelInfo) error {
	indexMu.Lock()
	defer indexMu.Unlock()
	return unsafeSaveIndex(idx)
}

func UpdateIndex(fn func(map[string]ModelInfo) error) error {
	indexMu.Lock()
	defer indexMu.Unlock()
	idx := unsafeLoadIndex()
	if err := fn(idx); err != nil {
		return err
	}
	return unsafeSaveIndex(idx)
}

type SearchResult struct {
	ID          string `json:"id"`
	Likes       int    `json:"likes"`
	Downloads   int    `json:"downloads"`
	PipelineTag string `json:"pipeline_tag"`
	Description string `json:"description"`
	HasGGUF     bool   `json:"has_gguf"`
	Size        int64  `json:"size"`
}

func isGated(v interface{}) bool {
	if v == nil {
		return false
	}
	switch val := v.(type) {
	case bool:
		return val
	case string:
		return val != "" && val != "false"
	}
	return false
}

func SearchModels(query string) ([]SearchResult, error) {
	// Auto-append GGUF to surface real GGUF conversion repos, not original model repos
	searchQuery := query
	if !strings.Contains(strings.ToLower(query), "gguf") {
		searchQuery = query + " GGUF"
	}
	apiURL := fmt.Sprintf("https://huggingface.co/api/models?search=%s&sort=likes&direction=-1&limit=20&full=true", url.QueryEscape(searchQuery))
	resp, err := APIClient.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("searching models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("search failed (HTTP %d)", resp.StatusCode)
	}

	var hfResults []struct {
		ID          string `json:"id"`
		Likes       int    `json:"likes"`
		Downloads   int    `json:"downloads"`
		PipelineTag string `json:"pipeline_tag"`
		Gated       interface{} `json:"gated"`
		Siblings    []struct {
			Filename string `json:"rfilename"`
		} `json:"siblings"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&hfResults); err != nil {
		return nil, fmt.Errorf("parsing search results: %w", err)
	}

	results := make([]SearchResult, 0, len(hfResults))
	var ggufIDs []string
	for _, r := range hfResults {
		if r.PipelineTag != "" && r.PipelineTag != "text-generation" && r.PipelineTag != "text-generation-instruct" && r.PipelineTag != "image-text-to-text" {
			continue
		}
		var hasGGUF, hasSafeTensor bool
		for _, s := range r.Siblings {
			if strings.HasSuffix(s.Filename, ".gguf") {
				hasGGUF = true
			}
			if strings.HasSuffix(s.Filename, ".safetensors") {
				hasSafeTensor = true
			}
		}
		// Skip repos that are primarily the original model format (safetensors)
		if !hasGGUF || hasSafeTensor {
			continue
		}
		// Skip gated models that require manual license acceptance (download will 401)
		if isGated(r.Gated) {
			continue
		}
		results = append(results, SearchResult{
			ID:          r.ID,
			Likes:       r.Likes,
			Downloads:   r.Downloads,
			PipelineTag: r.PipelineTag,
			HasGGUF:     true,
		})
		ggufIDs = append(ggufIDs, r.ID)
	}

	// Fetch sizes concurrently from individual model endpoints
	type sizeResult struct {
		idx  int
		size int64
	}
	sizeCh := make(chan sizeResult, len(ggufIDs))
	for i, id := range ggufIDs {
		go func(idx int, modelID string) {
			var total int64
			u := fmt.Sprintf("https://huggingface.co/api/models/%s", modelID)
			if resp, err := APIClient.Get(u); err == nil {
				defer resp.Body.Close()
				if resp.StatusCode == 200 {
					var detail struct {
						GGUF *struct {
							Total int64 `json:"total"`
						} `json:"gguf"`
					}
					if json.NewDecoder(resp.Body).Decode(&detail) == nil && detail.GGUF != nil {
						total = detail.GGUF.Total
					}
				}
			}
			sizeCh <- sizeResult{idx, total}
		}(i, id)
	}
	for range ggufIDs {
		sr := <-sizeCh
		if sr.idx < len(results) {
			results[sr.idx].Size = sr.size
		}
	}

	return results, nil
}

// RepoGGUFFile describes a single GGUF file (or grouped multi-part set) in a HF repo.
type RepoGGUFFile struct {
	Quant     string   `json:"quant"`
	Filename  string   `json:"filename"`
	Size      int64    `json:"size"`
	FileCount int      `json:"file_count"`
	Siblings  []string `json:"siblings"` // all filenames if multi-part
}

// knownQuantSuffixes lists quantization strings we recognize in filenames.
// Ordered by specificity (longer matches first) to avoid partial matches.
var knownQuantSuffixes = []string{
	"UD-Q3_K_S", "UD-Q3_K_M", "UD-Q3_K_L", "UD-Q4_K_M", "UD-Q4_K_S", "UD-Q5_K_M", "UD-Q5_K_S", "UD-Q6_K", "UD-Q8_0",
	"ARM-Q4_K_M", "ARM-Q4_K_S", "ARM-Q5_K_M", "ARM-Q5_K_S", "ARM-Q8_0",
	"IQ1_S", "IQ1_M", "IQ2_XXS", "IQ2_XS", "IQ2_S", "IQ2_M", "IQ3_XXS", "IQ3_S", "IQ4_NL", "IQ4_XS",
	"Q2_K", "Q3_K_S", "Q3_K_M", "Q3_K_L", "Q4_0", "Q4_1", "Q4_K_S", "Q4_K_M", "Q5_0", "Q5_1", "Q5_K_S", "Q5_K_M", "Q6_K", "Q8_0", "Q8_1", "Q8_K",
	"BF16", "F32", "F16",
}

// ListRepoGGUFFiles fetches the HF API for a repo and returns all GGUF files
// with parsed quantization, sizes, and multi-part grouping.
func ListRepoGGUFFiles(repo string) ([]RepoGGUFFile, error) {
	apiURL := fmt.Sprintf("https://huggingface.co/api/models/%s", repo)
	resp, err := APIClient.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("fetching repo %s: %w", repo, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("repo %s not found (HTTP %d)", repo, resp.StatusCode)
	}

	var data struct {
		Siblings []struct {
			Filename string `json:"rfilename"`
			Size     int64  `json:"size"`
		} `json:"siblings"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("parsing repo %s: %w", repo, err)
	}

	splitRe := regexp.MustCompile(`-(\d{5})-of-(\d{5})\.gguf$`)
	type candidate struct {
		filename string
		size     int64
		stem     string
		part     int
		total    int
		isSplit  bool
	}

	var files []candidate
	for _, s := range data.Siblings {
		if !strings.HasSuffix(s.Filename, ".gguf") {
			continue
		}
		fname := filepath.Base(s.Filename)
		if m := splitRe.FindStringSubmatch(fname); m != nil {
			part, _ := strconv.Atoi(m[1])
			total, _ := strconv.Atoi(m[2])
			files = append(files, candidate{
				filename: s.Filename,
				size:     s.Size,
				stem:     fname[:len(fname)-len(m[0])],
				part:     part,
				total:    total,
				isSplit:  true,
			})
		} else {
			files = append(files, candidate{
				filename: s.Filename,
				size:     s.Size,
				stem:     strings.TrimSuffix(fname, ".gguf"),
				isSplit:  false,
			})
		}
	}

	// Group multi-part files by their prefix stem
	type groupKey struct {
		stem  string
		total int
	}
	splitGroups := make(map[groupKey][]candidate)
	var singles []candidate

	for _, f := range files {
		if f.isSplit {
			key := groupKey{f.stem, f.total}
			splitGroups[key] = append(splitGroups[key], f)
		} else {
			singles = append(singles, f)
		}
	}

	// Parse quantization from a stem string.
	parseQuant := func(stem string) string {
		for _, q := range knownQuantSuffixes {
			if strings.HasSuffix(stem, q) || strings.HasSuffix(stem, "-"+q) {
				return q
			}
		}
		return ""
	}

	result := make([]RepoGGUFFile, 0, len(singles)+len(splitGroups))

	for _, f := range singles {
		quant := parseQuant(f.stem)
		result = append(result, RepoGGUFFile{
			Quant:     quant,
			Filename:  filepath.Base(f.filename),
			Size:      f.size,
			FileCount: 1,
			Siblings:  []string{f.filename},
		})
	}

	for key, parts := range splitGroups {
		var totalSize int64
		names := make([]string, len(parts))
		for _, p := range parts {
			totalSize += p.size
			names[p.part-1] = p.filename
		}
		quant := parseQuant(key.stem)
		result = append(result, RepoGGUFFile{
			Quant:     quant,
			Filename:  filepath.Base(parts[0].filename),
			Size:      totalSize,
			FileCount: key.total,
			Siblings:  names,
		})
	}

	return result, nil
}

// ScanModels scans the models directory for .gguf files not in the index and
// adds them. It always runs the scan (no throttling).
func ScanModels() {
	doScanModels()
}

// ScanModelsForce runs the scan and updates the throttle clock so subsequent
// throttled scans are suppressed. Used by the ?refresh=1 model-list query.
func ScanModelsForce() {
	scanMu.Lock()
	lastScanTime = time.Now()
	scanMu.Unlock()
	doScanModels()
}

// ResetScanThrottle clears the models-dir scan throttle clock so the next
// throttled scan runs immediately. Intended for tests.
func ResetScanThrottle() {
	scanMu.Lock()
	lastScanTime = time.Time{}
	scanMu.Unlock()
}

// scanModelsMaybe runs the scan at most once per scanInterval.
func scanModelsMaybe() {
	scanMu.Lock()
	if scanInterval > 0 && time.Since(lastScanTime) < scanInterval {
		scanMu.Unlock()
		return
	}
	lastScanTime = time.Now()
	scanMu.Unlock()
	doScanModels()
}

func doScanModels() {
	entries, err := os.ReadDir(ModelsDir())
	if err != nil {
		return
	}
	if err := UpdateIndex(func(idx map[string]ModelInfo) error {
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".gguf") || strings.HasSuffix(entry.Name(), ".part") {
				continue
			}
			path := filepath.Join(ModelsDir(), entry.Name())
			if fi, err := os.Stat(path); err != nil || fi.Size() == 0 {
				continue
			}
			// Handle split files (e.g. -00002-of-00004.gguf)
			splitRe := regexp.MustCompile(`-(\d{5})-of-(\d{5})\.gguf$`)
			if m := splitRe.FindStringSubmatch(entry.Name()); m != nil {
				// Only index the first part of a split
				if m[1] != "00001" {
					continue
				}
				// Key is the name without dir name artifacts
				base := entry.Name()
				base = splitRe.ReplaceAllString(base, "")
				// Strip quantization suffix for cleaner name
				base = strings.ToLower(StripQuantSuffix(base))
				base = strings.ReplaceAll(base, "_", "-")
				base = strings.TrimSuffix(base, ".gguf")
				fi, _ := os.Stat(path)
				info := ModelInfo{
					Name:     base,
					BlobPath: path,
					Source:   "local",
				}
				if fi != nil {
					info.Size = fi.Size()
				}
				if meta, err := readGGUFMetadata(path); err == nil && meta != nil {
					info.Architecture = meta.Architecture
					info.Quantization = meta.Quantization
					info.ContextLength = meta.ContextLength
					info.BlockCount = meta.BlockCount
				}
				// Sum sizes across all split parts
				if totalStr := m[2]; totalStr != "" {
					if totalParts, err := strconv.Atoi(totalStr); err == nil && totalParts > 1 {
						var totalSize int64
						for part := 1; part <= totalParts; part++ {
							pName := splitRe.ReplaceAllString(entry.Name(), fmt.Sprintf("-%05d-of-%05d.gguf", part, totalParts))
							pPath := filepath.Join(ModelsDir(), pName)
							if pfi, pErr := os.Stat(pPath); pErr == nil {
								totalSize += pfi.Size()
							}
						}
						if totalSize > 0 {
							info.Size = totalSize
						}
					}
				}
				short := strings.ToLower(base)
				info.ShortName = short
				if existing, exists := idx[base]; !exists {
					idx[base] = info
					slog.Info("scanned split model", "model", base, "arch", info.Architecture, "quant", info.Quantization, "size", FormatSize(info.Size))
				} else if _, err := os.Stat(existing.BlobPath); os.IsNotExist(err) {
					idx[base] = info
					slog.Info("replaced stale split model", "model", base, "arch", info.Architecture, "quant", info.Quantization, "size", FormatSize(info.Size))
				}
				continue
			}
			// Check if already indexed
			already := false
			for _, info := range idx {
				if info.BlobPath == path {
					already = true
					break
				}
			}
			if already {
				continue
			}
			fi, _ := os.Stat(path)
			info := ModelInfo{
				Name:     entry.Name(),
				BlobPath: path,
				Source:   "local",
			}
			if fi != nil {
				info.Size = fi.Size()
			}
			if meta, err := readGGUFMetadata(path); err == nil && meta != nil {
				info.Architecture = meta.Architecture
				info.Quantization = meta.Quantization
				info.ContextLength = meta.ContextLength
				info.BlockCount = meta.BlockCount
			}
			base := entry.Name()
			if strings.HasSuffix(base, ".gguf") {
				base = base[:len(base)-5]
			}
			// Normalize: underscores → hyphens for consistency
			info.Name = strings.ReplaceAll(base, "_", "-")
			// Generate a clean short name: strip the quantization tag FIRST
			// (it is underscore-separated — StripQuantSuffix understands both
			// separators), then normalize. Stripping after underscore→hyphen
			// normalization would never match (P5-T2).
			short := strings.ToLower(base)
			short = StripQuantSuffix(short)
			short = strings.ReplaceAll(short, "_", "-")
			short = strings.ReplaceAll(short, " ", "-")
			info.ShortName = short
			idx[base] = info
			slog.Info("scanned new model", "model", base, "short", short, "arch", info.Architecture, "quant", info.Quantization)
		}
		return nil
	}); err != nil {
		slog.Warn("could not save model index after scan", "error", err)
	}
}

func ListModels() ([]ModelInfo, error) {
	// Auto-discover new GGUF files (throttled — see scanInterval).
	scanModelsMaybe()

	// Copy data under read lock, then release before calling
	// populateModelInfo (which acquires a write lock).
	indexMu.RLock()
	idx := unsafeLoadIndex()
	var modelList []ModelInfo
	for _, info := range idx {
		if _, err := os.Stat(info.BlobPath); err == nil {
			info.Source = "local"
			modelList = append(modelList, info)
		}
	}
	indexMu.RUnlock()

	for i := range modelList {
		info := &modelList[i]
		// Skip the (expensive) GGUF header read when the index entry is
		// already complete (P4-T2).
		if info.Architecture != "" && info.Quantization != "" &&
			info.ContextLength > 0 && info.ShortName != "" {
			continue
		}
		populateModelInfo(info)
	}
	return modelList, nil
}

// ImagePythonPath returns the absolute path to the Python interpreter for image generation.
func ImagePythonPath() string {
	if p := os.Getenv("GOLLAMA_IMAGE_PYTHON"); p != "" {
		return p
	}
	return "/opt/image-api/.venv/bin/python"
}

// ImageAppPath returns the absolute path to the image generation server script.
func ImageAppPath() string {
	if p := os.Getenv("GOLLAMA_IMAGE_APP"); p != "" {
		return p
	}
	return "/opt/image-api/app.py"
}

// ImageModelCacheDir returns the path where diffusers models are cached.
func ImageModelCacheDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "huggingface", "hub")
}

// ImageModelCachePath returns the cache directory for a specific HF model ID.
func ImageModelCachePath(modelID string) string {
	folder := "models--" + strings.ReplaceAll(modelID, "/", "--")
	return filepath.Join(ImageModelCacheDir(), folder)
}

// IsImageModelCached checks if a diffusers model is already downloaded.
func IsImageModelCached(modelID string) bool {
	cachePath := ImageModelCachePath(modelID)
	fi, err := os.Stat(cachePath)
	if err != nil || !fi.IsDir() {
		return false
	}
	return true
}

// ImageModelCacheSize returns the total size of a cached model in bytes.
func ImageModelCacheSize(modelID string) int64 {
	cachePath := ImageModelCachePath(modelID)
	var total int64
	filepath.Walk(cachePath, func(path string, fi os.FileInfo, err error) error {
		if err != nil || fi == nil {
			return nil
		}
		if !fi.IsDir() {
			total += fi.Size()
		}
		return nil
	})
	return total
}

// ImageModelSearchResult describes an image model found on HF.
type ImageModelSearchResult struct {
	ID          string `json:"id"`
	Likes       int    `json:"likes"`
	Downloads   int    `json:"downloads"`
	PipelineTag string `json:"pipeline_tag"`
	Gated       bool   `json:"gated"`
	Size        int64  `json:"size"`
	License     string `json:"license,omitempty"`
}

// SearchImageModels searches HuggingFace for text-to-image models.
func SearchImageModels(query string) ([]ImageModelSearchResult, error) {
	if query == "" {
		query = "text-to-image"
	}
	apiURL := fmt.Sprintf("https://huggingface.co/api/models?search=%s&sort=likes&direction=-1&limit=20&full=true", url.QueryEscape(query))
	resp, err := APIClient.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("searching models: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("search failed (HTTP %d)", resp.StatusCode)
	}

	var hfResults []struct {
		ID          string      `json:"id"`
		Likes       int         `json:"likes"`
		Downloads   int         `json:"downloads"`
		PipelineTag string      `json:"pipeline_tag"`
		Gated       interface{} `json:"gated"`
		Siblings    []struct {
			Filename string `json:"rfilename"`
			Size     int64  `json:"size"`
		} `json:"siblings"`
		CardData struct {
			License string `json:"license"`
		} `json:"cardData"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&hfResults); err != nil {
		return nil, fmt.Errorf("parsing search results: %w", err)
	}

	results := make([]ImageModelSearchResult, 0, len(hfResults))
	for _, r := range hfResults {
		if r.PipelineTag != "text-to-image" && r.PipelineTag != "image-to-image" {
			continue
		}
		result := ImageModelSearchResult{
			ID:          r.ID,
			Likes:       r.Likes,
			Downloads:   r.Downloads,
			PipelineTag: r.PipelineTag,
			Gated:       isGated(r.Gated),
			License:     r.CardData.License,
		}
		for _, s := range r.Siblings {
			if result.Size < s.Size {
				result.Size = s.Size
			}
		}
		results = append(results, result)
	}
	return results, nil
}

// FreeDiskBytes returns free disk space in bytes for the given path's filesystem.
func FreeDiskBytes(path string) (uint64, error) {
	return freeDiskBytes(path)
}

func ResolveModelBlob(model string) (string, error) {
	idx := LoadIndex()
	// Exact match first
	if info, ok := idx[model]; ok {
		if _, err := os.Stat(info.BlobPath); err == nil {
			return info.BlobPath, nil
		}
	}
	// Fuzzy match — find by short name or substring (e.g. "gemma-4-12b"
	// matches short_name or full name). Deterministic (P2-T5): short-name
	// candidates win over substring candidates, and ties within a tier are
	// broken by lexicographic index name (the old code returned whichever
	// candidate map iteration hit first).
	lowerModel := strings.ToLower(model)
	var shortCands, substrCands []string
	for name, info := range idx {
		switch {
		case info.ShortName != "" && strings.EqualFold(info.ShortName, model):
			shortCands = append(shortCands, name)
		case strings.Contains(strings.ToLower(info.Name), lowerModel):
			substrCands = append(substrCands, name)
		}
	}
	for _, list := range [][]string{shortCands, substrCands} {
		sort.Strings(list)
		for _, name := range list {
			if _, err := os.Stat(idx[name].BlobPath); err == nil {
				return idx[name].BlobPath, nil
			}
		}
	}
	// Try as a direct file path
	if _, err := os.Stat(model); err == nil {
		return model, nil
	}
	return "", fmt.Errorf("model %q not found in index: %w", model, ErrNotFound)
}

func PullModel(ref string) error {
	return PullModelWithContext(context.Background(), ref)
}

func PullModelWithContext(ctx context.Context, ref string) error {
	return pullModelInternal(ctx, ref, nil, nil)
}

func PullModelWithProgress(ref string, progress io.Writer) error {
	return pullModelInternal(context.Background(), ref, nil, progress)
}

func PullModelWithCallback(ref string, fn ProgressFn) error {
	return pullModelInternal(context.Background(), ref, fn, nil)
}

func PullModelWithCallbackContext(ctx context.Context, ref string, fn ProgressFn) error {
	return pullModelInternal(ctx, ref, fn, nil)
}

// probeRemoteFileSize returns the remote size of url. It prefers an already
// known size, then a HEAD request, then a ranged GET and the Content-Range
// header. Returns 0 when unknown.
func probeRemoteFileSize(ctx context.Context, url string, known int64) int64 {
	if known > 0 {
		return known
	}
	headReq, _ := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if headResp, err := HTTPClient.Do(headReq); err == nil {
		remoteSize := headResp.ContentLength
		headResp.Body.Close()
		if remoteSize > 0 {
			return remoteSize
		}
	}
	rangeReq, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	rangeReq.Header.Set("Range", "bytes=0-0")
	if rangeResp, err := HTTPClient.Do(rangeReq); err == nil {
		defer rangeResp.Body.Close()
		cr := rangeResp.Header.Get("Content-Range")
		if cr != "" {
			parts := strings.Split(cr, "/")
			if len(parts) == 2 {
				if sz, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
					return sz
				}
			}
		}
	}
	return 0
}

// openPartialFile opens a download partial file for writing: appending when
// resuming, truncating when starting fresh.
func openPartialFile(path string, append bool) (*os.File, error) {
	flags := os.O_CREATE | os.O_WRONLY
	if append {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}
	return os.OpenFile(path, flags, 0644)
}

func pullModelInternal(ctx context.Context, ref string, fn ProgressFn, progress io.Writer) error {
	if !strings.HasPrefix(ref, "hf.co/") {
		ref = "hf.co/" + ref
	}

	parts := strings.SplitN(ref, ":", 2)
	modelID := strings.TrimPrefix(parts[0], "hf.co/")
	quant := "Q4_K_M"
	if len(parts) > 1 && parts[1] != "" {
		quant = parts[1]
	}

	apiURL := fmt.Sprintf("%s/api/models/%s", hfBaseURL, modelID)
	resp, err := APIClient.Get(apiURL)
	if err != nil {
		return fmt.Errorf("fetching model info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("model %s not found on HuggingFace (HTTP %d)", modelID, resp.StatusCode)
	}

	var modelData struct {
		Siblings []struct {
			Filename string `json:"rfilename"`
			Size     int64  `json:"size"`
		} `json:"siblings"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&modelData); err != nil {
		return fmt.Errorf("parsing model info: %w", err)
	}

	// Find matching GGUF files (handles multi-file splits)
	type sibling struct {
		Filename string `json:"rfilename"`
		Size     int64  `json:"size"`
	}
	var targetFiles []sibling
	splitPartRe := regexp.MustCompile(`-\d{5}-of-\d{5}$`)
	var minSegments int
	// Quant matching is case-insensitive: repos frequently use lowercase
	// filenames (e.g. q4_k_m.gguf) while users request the canonical
	// uppercase form (Q4_K_M). Comparing in uppercase avoids silently
	// falling back to "all GGUF files" when only the case differs.
	quantUpper := strings.ToUpper(quant)
	for _, s := range modelData.Siblings {
		if strings.HasSuffix(s.Filename, ".gguf") {
			fname := filepath.Base(s.Filename)
			stem := strings.TrimSuffix(fname, ".gguf")
			stem = splitPartRe.ReplaceAllString(stem, "")
			// Match quant as a suffix of the remaining stem (handles UD-Q3_K_S, IQ1_M, etc.)
			if strings.HasSuffix(strings.ToUpper(stem), quantUpper) || strings.HasSuffix(strings.ToUpper(stem), "-"+quantUpper) {
				segments := strings.Split(stem, "-")
				if len(segments) < minSegments || minSegments == 0 {
					minSegments = len(segments)
					targetFiles = nil
				}
				if len(segments) == minSegments {
					targetFiles = append(targetFiles, s)
				}
			}
		}
	}
	if len(targetFiles) == 0 {
		for _, s := range modelData.Siblings {
			if strings.HasSuffix(s.Filename, ".gguf") {
				targetFiles = append(targetFiles, s)
			}
		}
	}
	if len(targetFiles) == 0 {
		return fmt.Errorf("no GGUF file found in %s", modelID)
	}

	// Check for multi-file split pattern (e.g. model-00001-of-00004.gguf)
	splitRe := regexp.MustCompile(`-(\d{5})-of-(\d{5})\.gguf$`)
	firstMatch := splitRe.FindStringSubmatch(targetFiles[0].Filename)
	if firstMatch != nil {
		prefix := targetFiles[0].Filename[:len(targetFiles[0].Filename)-len(firstMatch[0])]
		totalParts, _ := strconv.Atoi(firstMatch[2])
		var resolved []sibling
		for i := 1; i <= totalParts; i++ {
			partName := fmt.Sprintf("%s-%05d-of-%05d.gguf", prefix, i, totalParts)
			found := false
			for _, s := range targetFiles {
				if s.Filename == partName {
					resolved = append(resolved, s)
					found = true
					break
				}
			}
			if !found {
				// Try full sibling list in case quant filter excluded a part
				for _, s := range modelData.Siblings {
					if s.Filename == partName {
						resolved = append(resolved, s)
						found = true
						break
					}
				}
			}
			if !found {
				return fmt.Errorf("missing split file: %s (part %d of %d)", partName, i, totalParts)
			}
		}
		targetFiles = resolved
	}

	modelName := fmt.Sprintf("hf.co/%s:%s", modelID, quant)

	EnsureDir(ModelsDir())

	// Check disk space for all parts combined
	var totalSize int64
	for _, f := range targetFiles {
		totalSize += f.Size
	}
	if totalSize > 0 {
		free, err := diskSpaceFn(ModelsDir())
		if err != nil {
			return fmt.Errorf("unable to check disk space: %w", err)
		} else {
			buffer := uint64(500 * 1024 * 1024)
			if uint64(totalSize) > buffer {
				buffer = uint64(totalSize) / 4
			}
			need := uint64(totalSize) + buffer
			if free < need {
				return fmt.Errorf("not enough disk space: need %s but only %s free — try a smaller quantization",
					FormatSize(int64(need)), FormatSize(int64(free)))
			}
		}
	}

	// Check if all parts already exist with correct sizes
	allExist := true
	for _, f := range targetFiles {
		fi, err := os.Stat(filepath.Join(ModelsDir(), filepath.Base(f.Filename)))
		if err != nil || (f.Size > 0 && fi.Size() != f.Size) {
			allExist = false
			break
		}
	}
	if allExist {
		firstDest := filepath.Join(ModelsDir(), filepath.Base(targetFiles[0].Filename))
		info := ModelInfo{Name: modelName, BlobPath: firstDest, ShortName: DeriveShortNameFromRepo(modelID)}
		if fi, err := os.Stat(firstDest); err == nil {
			info.Size = fi.Size()
		}
		if meta, err := readGGUFMetadata(firstDest); err == nil && meta != nil {
			info.Architecture = meta.Architecture
			info.Quantization = meta.Quantization
			info.ContextLength = meta.ContextLength
		}
		if err := UpdateIndex(func(idx map[string]ModelInfo) error {
			if _, exists := idx[modelName]; !exists {
				idx[modelName] = info
			}
			return nil
		}); err != nil {
			slog.Warn("could not re-index existing model", "model", modelName, "error", err)
		}
		slog.Info("model already exists, skipping download", "model", modelName)
		return ErrAlreadyExists
	}

	// Clean up stale index entry
	if err := UpdateIndex(func(idx map[string]ModelInfo) error {
		delete(idx, modelName)
		return nil
	}); err != nil {
		slog.Warn("could not clear stale index entry", "model", modelName, "error", err)
	}

	// Track files we've started downloading so we can clean up on cancel
	// Track partial files we've started so we can clean them up on cancel.
	var pendingParts []string
	defer func() {
		if ctx.Err() != nil {
			for _, path := range pendingParts {
				os.Remove(path)
			}
		}
	}()

	for i, f := range targetFiles {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		dest := filepath.Join(ModelsDir(), filepath.Base(f.Filename))
		part := dest + ".part"
		downloadURL := fmt.Sprintf("%s/%s/resolve/main/%s", hfBaseURL, modelID, f.Filename)

		remoteSize := probeRemoteFileSize(ctx, downloadURL, f.Size)

		// Skip if file already exists with correct size
		if remoteSize > 0 {
			if fi, err := os.Stat(dest); err == nil && fi.Size() == remoteSize {
				fmt.Printf("[%d/%d] %s already exists, skipping\n", i+1, len(targetFiles), f.Filename)
				os.Remove(part) // clean up any stale partial file
				continue
			}
		}

		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Resume from an existing partial file, if any.
		var done int64
		if fi, err := os.Stat(part); err == nil {
			done = fi.Size()
			if remoteSize > 0 {
				if done > remoteSize {
					slog.Warn("partial file larger than remote file, restarting",
						"file", part, "local", FormatSize(done), "remote", FormatSize(remoteSize))
					os.Remove(part)
					done = 0
				} else if done == remoteSize {
					// A previous run finished the bytes but died before the rename.
					fmt.Printf("[%d/%d] %s partial already complete — finalizing\n", i+1, len(targetFiles), f.Filename)
					if err := os.Rename(part, dest); err != nil {
						return fmt.Errorf("finalizing %s: %w", f.Filename, err)
					}
					continue
				}
			}
		}

		if remoteSize > 0 {
			fmt.Printf("[%d/%d] Downloading %s (%s)\n", i+1, len(targetFiles), f.Filename, FormatSize(remoteSize))
		} else {
			fmt.Printf("[%d/%d] Downloading %s\n", i+1, len(targetFiles), f.Filename)
		}
		if done > 0 {
			fmt.Printf("  resuming from %s\n", FormatSize(done))
		}

		out, err := openPartialFile(part, done > 0)
		if err != nil {
			return fmt.Errorf("creating partial file for %s: %w", f.Filename, err)
		}
		pendingParts = append(pendingParts, part)

		dlReq, _ := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
		if done > 0 {
			dlReq.Header.Set("Range", fmt.Sprintf("bytes=%d-", done))
		}
		dlResp, err := HTTPClient.Do(dlReq)
		if err != nil {
			out.Close()
			os.Remove(part)
			return fmt.Errorf("downloading %s: %w", f.Filename, err)
		}

		if dlResp.StatusCode == 200 && done > 0 {
			// Server ignored our Range request — start over from zero.
			dlResp.Body.Close()
			out.Close()
			out, err = openPartialFile(part, false)
			if err != nil {
				return fmt.Errorf("creating partial file for %s: %w", f.Filename, err)
			}
			done = 0
		} else if dlResp.StatusCode != 200 && dlResp.StatusCode != 206 {
			out.Close()
			dlResp.Body.Close()
			os.Remove(part)
			return fmt.Errorf("download failed for %s (HTTP %d)", f.Filename, dlResp.StatusCode)
		}

		dlSize := remoteSize
		if dlSize == 0 && dlResp.ContentLength > 0 {
			dlSize = dlResp.ContentLength
		}

		pr := &ProgressReader{
			Reader:     dlResp.Body,
			Total:      dlSize,
			Done:       done,
			Name:       "▸",
			Start:      time.Now(),
			Output:     progress,
			ProgressFn: fn,
			Part:       i + 1,
			TotalParts: len(targetFiles),
			TTY:        progressOutputTTY(progress),
		}
		_, cpErr := io.Copy(out, pr)
		out.Close()
		dlResp.Body.Close()
		if cpErr != nil {
			os.Remove(part)
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("downloading %s: %w", f.Filename, cpErr)
		}

		// Verify final size (when known) before promoting the partial file.
		if fi, err := os.Stat(part); err == nil && remoteSize > 0 && fi.Size() != remoteSize {
			os.Remove(part)
			return fmt.Errorf("size mismatch for %s: got %s, expected %s",
				f.Filename, FormatSize(fi.Size()), FormatSize(remoteSize))
		}

		if err := os.Rename(part, dest); err != nil {
			return fmt.Errorf("finalizing %s: %w", f.Filename, err)
		}
	}

	// Index using the first split file (llama-server discovers the rest by naming convention)
	firstDest := filepath.Join(ModelsDir(), filepath.Base(targetFiles[0].Filename))
	info := ModelInfo{
		Name:      modelName,
		ShortName: DeriveShortNameFromRepo(modelID),
		BlobPath: firstDest,
		Size:     totalSize,
	}
	if fi, err := os.Stat(firstDest); err == nil {
		info.Size = fi.Size()
	}
	populateModelInfo(&info)
	if err := UpdateIndex(func(idx map[string]ModelInfo) error {
		idx[modelName] = info
		return nil
	}); err != nil {
		return fmt.Errorf("model downloaded but could not be indexed: %w", err)
	}

	slog.Info("model downloaded", "model", modelName, "size", FormatSize(totalSize), "files", len(targetFiles), "dest", firstDest)
	fmt.Printf("Downloaded %s (%s, %d files)\n", modelName, FormatSize(totalSize), len(targetFiles))
	return nil
}
