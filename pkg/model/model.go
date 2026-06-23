package model

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
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

type Config struct {
	DefaultFlags  []string `json:"default_flags"`
	ProxyDefaults []string `json:"proxy_defaults"` // flags for auto-launched instances; falls back to default_flags
	IdleTTL       int      `json:"idle_ttl"`       // minutes; 0 = disabled
}

func ConfigFile() string {
	return filepath.Join(GollamaDir(), "config.json")
}

func IsStandaloneFlag(name string) bool {
	return standaloneFlags[name]
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
	flags := []string{"--host", "0.0.0.0", "--ctx-size", "2048", "--flash-attn", "on", "--temp", "0.7"}
	return &Config{DefaultFlags: flags, IdleTTL: 30}
}

func sanitizeFlags(flags []string) []string {
	clean := make([]string, 0, len(flags))
	for _, f := range flags {
		if strings.HasPrefix(f, "--") {
			clean = append(clean, f)
		} else if len(clean) > 0 {
			last := clean[len(clean)-1]
			if !strings.HasPrefix(last, "--") {
				// Consecutive values without a preceding flag key — skip
				continue
			}
			if standaloneFlags[last] {
				// Standalone boolean flag — skip its orphaned value
				continue
			}
			clean = append(clean, f)
		} else {
			// Leading value without a flag key — skip
			continue
		}
	}
	return clean
}

func LoadConfig() *Config {
	EnsureDir(GollamaDir())
	data, err := os.ReadFile(ConfigFile())
	if err != nil {
		cfg := DefaultConfig()
		SaveConfig(cfg)
		return cfg
	}
	var cfg Config
	if json.Unmarshal(data, &cfg) != nil || cfg.DefaultFlags == nil {
		cfg = *DefaultConfig()
		SaveConfig(&cfg)
	}
	cfg.DefaultFlags = sanitizeFlags(cfg.DefaultFlags)
	cfg.ProxyDefaults = sanitizeFlags(cfg.ProxyDefaults)
	return &cfg
}

func (c *Config) ProxyFlags() []string {
	if len(c.ProxyDefaults) > 0 {
		return c.ProxyDefaults
	}
	return c.DefaultFlags
}

func SaveConfig(cfg *Config) {
	cfg.DefaultFlags = sanitizeFlags(cfg.DefaultFlags)
	cfg.ProxyDefaults = sanitizeFlags(cfg.ProxyDefaults)
	data, _ := json.MarshalIndent(cfg, "", "  ")
	os.WriteFile(ConfigFile(), data, 0644)
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

func IndexFile() string {
	return filepath.Join(GollamaDir(), "index.json")
}

func VersionFile() string {
	return filepath.Join(GollamaDir(), "llama-server-version.txt")
}

func BackendFile() string {
	return filepath.Join(GollamaDir(), "llama-server-backend.txt")
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
	} else {
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
	}
	if err == io.EOF && pr.ProgressFn == nil {
		fmt.Fprintln(pr.output())
	}
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

// HTTPClient is a shared HTTP client with a custom User-Agent.
var HTTPClient = &http.Client{
	Transport: &userAgentTransport{
		next: http.DefaultTransport,
	},
}

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

func unsafeSaveIndex(idx map[string]ModelInfo) {
	data, _ := json.MarshalIndent(idx, "", "  ")
	tmp := IndexFile() + ".tmp"
	os.WriteFile(tmp, data, 0644)
	os.Rename(tmp, IndexFile())
}

func LoadIndex() map[string]ModelInfo {
	indexMu.RLock()
	defer indexMu.RUnlock()
	return unsafeLoadIndex()
}

func SaveIndex(idx map[string]ModelInfo) {
	indexMu.Lock()
	defer indexMu.Unlock()
	unsafeSaveIndex(idx)
}

func UpdateIndex(fn func(map[string]ModelInfo) error) error {
	indexMu.Lock()
	defer indexMu.Unlock()
	idx := unsafeLoadIndex()
	if err := fn(idx); err != nil {
		return err
	}
	unsafeSaveIndex(idx)
	return nil
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
	resp, err := HTTPClient.Get(apiURL)
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
			if resp, err := HTTPClient.Get(u); err == nil {
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

// ScanModels scans the models directory for .gguf files not in the index and adds them.
func ScanModels() {
	entries, err := os.ReadDir(ModelsDir())
	if err != nil {
		return
	}
	UpdateIndex(func(idx map[string]ModelInfo) error {
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".gguf") {
				continue
			}
			path := filepath.Join(ModelsDir(), entry.Name())
			if _, err := os.Stat(path); err != nil {
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
			info.Name = base
			// Generate a clean short name: lowercase, underscores→hyphens, strip quants
			short := strings.ToLower(base)
			short = strings.ReplaceAll(short, "_", "-")
			short = strings.ReplaceAll(short, " ", "-")
			// Strip trailing quant pattern like -iq4_xs, -q4_k_m etc.
			quantRe := regexp.MustCompile(`(?i)-[iqbf][qkbf][0-9]_[slmx](_[slmx])?$`)
			short = quantRe.ReplaceAllString(short, "")
			info.ShortName = short
			idx[base] = info
			log.Printf("scanned new model: %s (short=%s, arch=%s, quant=%s)", base, short, info.Architecture, info.Quantization)
		}
		return nil
	})
}

func ListModels() ([]ModelInfo, error) {
	// Auto-discover new GGUF files
	ScanModels()

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
		populateModelInfo(&modelList[i])
	}
	return modelList, nil
}

func ResolveModelBlob(model string) (string, error) {
	idx := LoadIndex()
	// Exact match first
	if info, ok := idx[model]; ok {
		if _, err := os.Stat(info.BlobPath); err == nil {
			return info.BlobPath, nil
		}
	}
	// Fuzzy match — find by short name or substring (e.g. "gemma-4-12b" matches short_name or full name)
	lowerModel := strings.ToLower(model)
	for _, info := range idx {
		match := strings.ToLower(info.ShortName) == lowerModel ||
			strings.Contains(strings.ToLower(info.Name), lowerModel)
		if match {
			if _, err := os.Stat(info.BlobPath); err == nil {
				return info.BlobPath, nil
			}
		}
	}
	// Try as a direct file path
	if _, err := os.Stat(model); err == nil {
		return model, nil
	}
	return "", fmt.Errorf("model %q not found in index", model)
}

func PullModel(ref string) error {
	return PullModelWithProgress(ref, nil)
}

func PullModelWithProgress(ref string, progress io.Writer) error {
	return pullModelInternal(ref, nil, progress)
}

func PullModelWithCallback(ref string, fn ProgressFn) error {
	return pullModelInternal(ref, fn, nil)
}

func pullModelInternal(ref string, fn ProgressFn, progress io.Writer) error {
	if !strings.HasPrefix(ref, "hf.co/") {
		ref = "hf.co/" + ref
	}

	parts := strings.SplitN(ref, ":", 2)
	modelID := strings.TrimPrefix(parts[0], "hf.co/")
	quant := "Q4_K_M"
	if len(parts) > 1 && parts[1] != "" {
		quant = parts[1]
	}

	apiURL := fmt.Sprintf("https://huggingface.co/api/models/%s", modelID)
	resp, err := HTTPClient.Get(apiURL)
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
	for _, s := range modelData.Siblings {
		if strings.HasSuffix(s.Filename, ".gguf") {
			fname := filepath.Base(s.Filename)
			stem := strings.TrimSuffix(fname, ".gguf")
			stem = splitPartRe.ReplaceAllString(stem, "")
			segments := strings.Split(stem, "-")
			if len(segments) > 0 && segments[len(segments)-1] == quant {
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
		free, err := freeDiskBytes(ModelsDir())
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
		UpdateIndex(func(idx map[string]ModelInfo) error {
			if _, exists := idx[modelName]; !exists {
				idx[modelName] = info
			}
			return nil
		})
		log.Printf("model %s already exists, skipping download", modelName)
		return fmt.Errorf("already_exists")
	}

	// Clean up stale index entry
	UpdateIndex(func(idx map[string]ModelInfo) error {
		delete(idx, modelName)
		return nil
	})

	for i, f := range targetFiles {
		dest := filepath.Join(ModelsDir(), filepath.Base(f.Filename))
		downloadURL := fmt.Sprintf("https://huggingface.co/%s/resolve/main/%s", modelID, f.Filename)

		remoteSize := f.Size
		if remoteSize <= 0 {
			if headResp, headErr := HTTPClient.Head(downloadURL); headErr == nil {
				remoteSize = headResp.ContentLength
				headResp.Body.Close()
			}
		}
		if remoteSize <= 0 {
			rangeReq, _ := http.NewRequest("GET", downloadURL, nil)
			rangeReq.Header.Set("Range", "bytes=0-0")
			if rangeResp, rangeErr := HTTPClient.Do(rangeReq); rangeErr == nil {
				cr := rangeResp.Header.Get("Content-Range")
				if cr != "" {
					parts := strings.Split(cr, "/")
					if len(parts) == 2 {
						if sz, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
							remoteSize = sz
						}
					}
				}
				rangeResp.Body.Close()
			}
		}
		// Skip if file already exists with correct size
		if remoteSize > 0 {
			if fi, err := os.Stat(dest); err == nil && fi.Size() == remoteSize {
				fmt.Printf("[%d/%d] %s already exists, skipping\n", i+1, len(targetFiles), f.Filename)
				continue
			}
		}

		if remoteSize > 0 {
			fmt.Printf("[%d/%d] Downloading %s (%s)\n", i+1, len(targetFiles), f.Filename, FormatSize(remoteSize))
		} else {
			fmt.Printf("[%d/%d] Downloading %s\n", i+1, len(targetFiles), f.Filename)
		}

		out, err := os.Create(dest)
		if err != nil {
			return fmt.Errorf("creating file %s: %w", f.Filename, err)
		}

		dlResp, err := HTTPClient.Get(downloadURL)
		if err != nil {
			out.Close()
			os.Remove(dest)
			return fmt.Errorf("downloading %s: %w", f.Filename, err)
		}

		if dlResp.StatusCode != 200 {
			out.Close()
			dlResp.Body.Close()
			os.Remove(dest)
			return fmt.Errorf("download failed for %s (HTTP %d)", f.Filename, dlResp.StatusCode)
		}

		dlSize := f.Size
		if dlSize == 0 && dlResp.ContentLength > 0 {
			dlSize = dlResp.ContentLength
		}

		pr := &ProgressReader{
			Reader:     dlResp.Body,
			Total:      dlSize,
			Name:       "▸",
			Start:      time.Now(),
			Output:     progress,
			ProgressFn: fn,
			Part:       i + 1,
			TotalParts: len(targetFiles),
		}
		_, cpErr := io.Copy(out, pr)
		out.Close()
		dlResp.Body.Close()
		if cpErr != nil {
			os.Remove(dest)
			return fmt.Errorf("downloading %s: %w", f.Filename, cpErr)
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
	UpdateIndex(func(idx map[string]ModelInfo) error {
		idx[modelName] = info
		return nil
	})

	log.Printf("model downloaded: %s (%s, %d files) → %s", modelName, FormatSize(totalSize), len(targetFiles), firstDest)
	fmt.Printf("Downloaded %s (%s, %d files)\n", modelName, FormatSize(totalSize), len(targetFiles))
	return nil
}
