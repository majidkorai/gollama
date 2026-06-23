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
	expected := []string{"--host", "0.0.0.0", "--ctx-size", "2048", "--flash-attn", "on", "--temp", "0.7"}
	for i, f := range expected {
		if i >= len(cfg.DefaultFlags) || cfg.DefaultFlags[i] != f {
			t.Errorf("DefaultFlags[%d] = %q, want %q", i, cfg.DefaultFlags[i], f)
		}
	}
}

func TestConfigFile(t *testing.T) {
	path := ConfigFile()
	if !strings.HasSuffix(path, "config.json") {
		t.Errorf("ConfigFile() = %q, should end with config.json", path)
	}
	if !filepath.IsAbs(path) {
		t.Errorf("ConfigFile() = %q, should be absolute", path)
	}
}

func checkSuffix(t *testing.T, path, suffix string) {
	t.Helper()
	if !strings.HasSuffix(path, suffix) {
		t.Errorf("%q should end with %s", path, suffix)
	}
}

func TestGollamaDir(t *testing.T) {
	checkSuffix(t, GollamaDir(), ".gollama")
}

func TestModelsDir(t *testing.T) {
	checkSuffix(t, ModelsDir(), "models")
}

func TestBinDir(t *testing.T) {
	checkSuffix(t, BinDir(), "bin")
}

func TestIndexFile(t *testing.T) {
	checkSuffix(t, IndexFile(), "index.json")
}

func TestBackendFile(t *testing.T) {
	checkSuffix(t, BackendFile(), "llama-server-backend.txt")
}

func TestVersionFile(t *testing.T) {
	checkSuffix(t, VersionFile(), "llama-server-version.txt")
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
