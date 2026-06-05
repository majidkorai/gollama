package model

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

type ModelInfo struct {
	Name          string `json:"name"`
	BlobPath      string `json:"blob_path"`
	Size          int64  `json:"size"`
	Architecture  string `json:"architecture,omitempty"`
	Quantization  string `json:"quantization,omitempty"`
	ContextLength uint64 `json:"context_length,omitempty"`
	Source        string `json:"source,omitempty"`
}

type Preset struct {
	Name  string   `json:"name"`
	Model string   `json:"model"`
	Port  int      `json:"port"`
	Flags []string `json:"flags"`
}

type Config struct {
	DefaultFlags []string `json:"default_flags"`
}

func ConfigFile() string {
	return filepath.Join(GollamaDir(), "config.json")
}

func DefaultConfig() *Config {
	// CPU-safe defaults that work on laptops and low-RAM systems
	flags := []string{"--ctx-size", "2048", "--flash-attn", "on", "--reasoning-max-tokens", "2048"}
	return &Config{DefaultFlags: flags}
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
	return &cfg
}

func SaveConfig(cfg *Config) {
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
	// Returns whether a GPU is available and recommended --n-gpu-layers count.
	switch runtime.GOOS {
	case "linux":
		if _, err := os.Stat("/proc/driver/nvidia/version"); err == nil {
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

type ProgressReader struct {
	Reader io.Reader
	Total  int64
	Done   int64
	Start  time.Time
	Name   string
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
	if pr.Total > 0 {
		pct := float64(pr.Done) * 100 / float64(pr.Total)
		fmt.Fprintf(os.Stderr, "\r  %s  %.1f%%  (%s / %s)  %s    ",
			pr.Name, pct, FormatSize(pr.Done), FormatSize(pr.Total), speed)
	} else {
		fmt.Fprintf(os.Stderr, "\r  %s  %s  %s       ",
			pr.Name, FormatSize(pr.Done), speed)
	}
	if err == io.EOF {
		fmt.Fprintln(os.Stderr)
	}
	return n, err
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

func ListModels() ([]ModelInfo, error) {
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
	if info, ok := idx[model]; ok {
		if _, err := os.Stat(info.BlobPath); err == nil {
			return info.BlobPath, nil
		}
		UpdateIndex(func(idx map[string]ModelInfo) error {
			delete(idx, model)
			return nil
		})
	}
	if _, err := os.Stat(model); err == nil {
		return model, nil
	}
	return model, nil
}

func PullModel(ref string) error {
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

	var targetFile string
	var targetSize int64
	for _, s := range modelData.Siblings {
		if strings.HasSuffix(s.Filename, ".gguf") && strings.Contains(s.Filename, quant) {
			targetFile = s.Filename
			targetSize = s.Size
			break
		}
	}
	if targetFile == "" {
		for _, s := range modelData.Siblings {
			if strings.HasSuffix(s.Filename, ".gguf") {
				targetFile = s.Filename
				targetSize = s.Size
				break
			}
		}
	}
	if targetFile == "" {
		return fmt.Errorf("no GGUF file found in %s", modelID)
	}

	EnsureDir(ModelsDir())
	dest := filepath.Join(ModelsDir(), filepath.Base(targetFile))

	if targetSize > 0 {
		free, err := freeDiskBytes(ModelsDir())
		if err == nil {
			need := uint64(targetSize) + 500*1024*1024
			if free < need {
				return fmt.Errorf("not enough disk space: need %s but only %s free — try a smaller quantization",
					FormatSize(int64(need)), FormatSize(int64(free)))
			}
		}
	}

	downloadURL := fmt.Sprintf("https://huggingface.co/%s/resolve/main/%s", modelID, targetFile)
	if targetSize > 0 {
		fmt.Printf("Downloading %s (%s)\n", targetFile, FormatSize(targetSize))
	} else {
		fmt.Printf("Downloading %s\n", targetFile)
	}

	out, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer out.Close()

	dlResp, err := HTTPClient.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("downloading: %w", err)
	}
	defer dlResp.Body.Close()

	if dlResp.StatusCode != 200 {
		os.Remove(dest)
		return fmt.Errorf("download failed (HTTP %d)", dlResp.StatusCode)
	}

	pr := &ProgressReader{
		Reader: dlResp.Body,
		Total:  targetSize,
		Name:   "▸",
		Start:  time.Now(),
	}
	written, err := io.Copy(out, pr)
	if err != nil {
		os.Remove(dest)
		return fmt.Errorf("downloading: %w", err)
	}

	modelName := fmt.Sprintf("hf.co/%s:%s", modelID, quant)
	info := ModelInfo{
		Name:     modelName,
		BlobPath: dest,
		Size:     written,
	}
	populateModelInfo(&info)
	UpdateIndex(func(idx map[string]ModelInfo) error {
		idx[modelName] = info
		return nil
	})

	log.Printf("model downloaded: %s (%s) → %s", modelName, FormatSize(written), dest)
	fmt.Printf("Downloaded %s (%s)\n", modelName, FormatSize(written))
	return nil
}
