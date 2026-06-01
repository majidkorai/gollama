package model

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type ModelInfo struct {
	Name         string `json:"name"`
	BlobPath     string `json:"blob_path"`
	Size         int64  `json:"size"`
	Architecture string `json:"architecture,omitempty"`
	Quantization string `json:"quantization,omitempty"`
	Source       string `json:"source,omitempty"`
}

type Preset struct {
	Name  string   `json:"name"`
	Model string   `json:"model"`
	Port  int      `json:"port"`
	Flags []string `json:"flags"`
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
		fmt.Printf("\r  %s  %.1f%%  (%s / %s)  %s    ",
			pr.Name, pct, FormatSize(pr.Done), FormatSize(pr.Total), speed)
	} else {
		fmt.Printf("\r  %s  %s  %s       ",
			pr.Name, FormatSize(pr.Done), speed)
	}
	if err == io.EOF {
		fmt.Println()
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

func LoadIndex() map[string]ModelInfo {
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

func SaveIndex(idx map[string]ModelInfo) {
	data, _ := json.MarshalIndent(idx, "", "  ")
	os.WriteFile(IndexFile(), data, 0644)
}

func ListModels() ([]ModelInfo, error) {
	var models []ModelInfo
	idx := LoadIndex()
	for _, info := range idx {
		if _, err := os.Stat(info.BlobPath); err == nil {
			info.Source = "local"
			models = append(models, info)
		}
	}
	return models, nil
}

func ResolveModelBlob(model string) (string, error) {
	idx := LoadIndex()
	if info, ok := idx[model]; ok {
		if _, err := os.Stat(info.BlobPath); err == nil {
			return info.BlobPath, nil
		}
		delete(idx, model)
		SaveIndex(idx)
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
	resp, err := http.Get(apiURL)
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

	downloadURL := fmt.Sprintf("https://huggingface.co/%s/resolve/main/%s", modelID, targetFile)
	fmt.Printf("Downloading %s (%s)\n", targetFile, FormatSize(targetSize))

	out, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer out.Close()

	dlResp, err := http.Get(downloadURL)
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

	idx := LoadIndex()
	modelName := fmt.Sprintf("hf.co/%s:%s", modelID, quant)
	idx[modelName] = ModelInfo{
		Name:     modelName,
		BlobPath: dest,
		Size:     written,
	}
	SaveIndex(idx)

	log.Printf("model downloaded: %s (%s) → %s", modelName, FormatSize(written), dest)
	fmt.Printf("Downloaded %s (%s)\n", modelName, FormatSize(written))
	return nil
}
