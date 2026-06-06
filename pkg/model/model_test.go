package model

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFormatSize(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0 B"},
		{1, "1 B"},
		{512, "512 B"},
		{1024, "1024 B"},
		{1048576, "1.0 MB"},
		{2097152, "2.0 MB"},
		{1073741824, "1.0 GB"},
		{1610612736, "1.5 GB"},
		{3221225472, "3.0 GB"},
	}
	for _, tt := range tests {
		got := FormatSize(tt.input)
		if got != tt.want {
			t.Errorf("FormatSize(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg == nil {
		t.Fatal("DefaultConfig() returned nil")
	}
	if len(cfg.DefaultFlags) == 0 {
		t.Fatal("DefaultFlags is empty")
	}
	expected := []string{"--host", "0.0.0.0", "--ctx-size", "2048", "--flash-attn", "on", "--temp", "0.7", "--repeat-penalty", "1.1"}
	for i, f := range expected {
		if i >= len(cfg.DefaultFlags) || cfg.DefaultFlags[i] != f {
			t.Errorf("DefaultFlags[%d] = %q, want %q", i, cfg.DefaultFlags[i], f)
		}
	}
}

func TestConfigFile(t *testing.T) {
	path := ConfigFile()
	if !strings.HasSuffix(path, "/config.json") {
		t.Errorf("ConfigFile() = %q, should end with /config.json", path)
	}
	if !strings.HasPrefix(path, "/") {
		t.Errorf("ConfigFile() = %q, should be absolute", path)
	}
}

func TestGollamaDir(t *testing.T) {
	dir := GollamaDir()
	if !strings.HasSuffix(dir, ".gollama") {
		t.Errorf("GollamaDir() = %q, should end with .gollama", dir)
	}
}

func TestModelsDir(t *testing.T) {
	dir := ModelsDir()
	if !strings.HasSuffix(dir, "models") {
		t.Errorf("ModelsDir() = %q, should end with models", dir)
	}
}

func TestBinDir(t *testing.T) {
	dir := BinDir()
	if !strings.HasSuffix(dir, "bin") {
		t.Errorf("BinDir() = %q, should end with bin", dir)
	}
}

func TestIndexFile(t *testing.T) {
	path := IndexFile()
	if !strings.HasSuffix(path, "index.json") {
		t.Errorf("IndexFile() = %q, should end with index.json", path)
	}
}

func TestBackendFile(t *testing.T) {
	path := BackendFile()
	if !strings.HasSuffix(path, "llama-server-backend.txt") {
		t.Errorf("BackendFile() = %q, should end with llama-server-backend.txt", path)
	}
}

func TestVersionFile(t *testing.T) {
	path := VersionFile()
	if !strings.HasSuffix(path, "llama-server-version.txt") {
		t.Errorf("VersionFile() = %q, should end with llama-server-version.txt", path)
	}
}

func TestEnsureDir(t *testing.T) {
	tmp := t.TempDir()
	testDir := filepath.Join(tmp, "test", "nested", "dirs")
	EnsureDir(testDir)
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		t.Errorf("EnsureDir() did not create %s", testDir)
	}
}

func TestProgressReader(t *testing.T) {
	var buf bytes.Buffer
	for i := 0; i < 100; i++ {
		buf.WriteByte(byte(i))
	}
	pr := &ProgressReader{
		Reader: &buf,
		Total:  100,
		Name:   "test",
		Start:  time.Now(),
	}
	read, err := io.ReadAll(pr)
	if err != nil {
		t.Fatalf("ProgressReader.ReadAll() error: %v", err)
	}
	if len(read) != 100 {
		t.Errorf("ProgressReader read %d bytes, want 100", len(read))
	}
	if pr.Done != 100 {
		t.Errorf("ProgressReader.Done = %d, want 100", pr.Done)
	}
}

func TestProgressReaderNoTotal(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteString("hello world")
	pr := &ProgressReader{
		Reader: &buf,
		Total:  0,
		Name:   "test",
		Start:  time.Now(),
	}
	read, err := io.ReadAll(pr)
	if err != nil {
		t.Fatalf("ProgressReader.ReadAll() error: %v", err)
	}
	if string(read) != "hello world" {
		t.Errorf("ProgressReader read %q, want %q", string(read), "hello world")
	}
}
