package model

import (
	"bytes"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestMain disables models-dir scan throttling so tests always see a fresh
// scan regardless of run order or timing within the shared test process.
func TestMain(m *testing.M) {
	scanInterval = 0
	os.Exit(m.Run())
}

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
	// v3.8.0: loopback by default (was 0.0.0.0) — the API token is the gate
	// for any LAN exposure (--listen 0.0.0.0).
	expected := []string{"--host", "127.0.0.1", "--ctx-size", "2048", "--flash-attn", "on", "--temp", "0.7"}
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

func TestLoadTimeout(t *testing.T) {
	if got := LoadTimeout(); got != 5*time.Minute {
		t.Fatalf("default = %v, want 5m", got)
	}
	t.Setenv("GOLLAMA_MODEL_LOAD_TIMEOUT", "42")
	if got := LoadTimeout(); got != 42*time.Second {
		t.Fatalf("env override = %v, want 42s", got)
	}
	t.Setenv("GOLLAMA_MODEL_LOAD_TIMEOUT", "garbage")
	if got := LoadTimeout(); got != 5*time.Minute {
		t.Fatalf("garbage env = %v, want 5m", got)
	}
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

func TestGenerateAPIToken(t *testing.T) {
	tok, err := GenerateAPIToken()
	if err != nil {
		t.Fatalf("GenerateAPIToken: %v", err)
	}
	if len(tok) != 64 {
		t.Fatalf("token length = %d, want 64 (32 bytes hex)", len(tok))
	}
	if _, err := hex.DecodeString(tok); err != nil {
		t.Fatalf("token is not valid hex: %v", err)
	}
	tok2, _ := GenerateAPIToken()
	if tok == tok2 {
		t.Fatal("two generated tokens are identical")
	}
}

// setTestHome isolates the gollama dir for a test on every platform.
// os.UserHomeDir reads HOME on Unix but USERPROFILE on Windows, so we set
// both — otherwise Windows tests share one real dir and the
// api-token-generated marker leaks between tests.
func setTestHome(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
}

func TestEnsureAPITokenFreshInstall(t *testing.T) {
	setTestHome(t)
	tok, generated := EnsureAPIToken()
	if !generated {
		t.Fatal("fresh install should generate a token")
	}
	if len(tok) != 64 {
		t.Fatalf("generated token length = %d, want 64", len(tok))
	}
	cfg := LoadConfig()
	if cfg.APIToken != tok {
		t.Fatalf("config token = %q, want persisted %q", cfg.APIToken, tok)
	}
	// Second call is stable — same token, not regenerated.
	tok2, generated2 := EnsureAPIToken()
	if generated2 || tok2 != tok {
		t.Fatalf("second call: generated=%v token=%q, want stable %q", generated2, tok2, tok)
	}
}

func TestEnsureAPITokenUpgradedConfig(t *testing.T) {
	setTestHome(t)
	// A pre-v3.8 config has no api_token key at all.
	raw := `{"default_flags":["--ctx-size","512"],"proxy_defaults":[],"profiles":{},"idle_ttl":0}`
	if err := os.MkdirAll(GollamaDir(), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ConfigFile(), []byte(raw), 0644); err != nil {
		t.Fatal(err)
	}
	tok, generated := EnsureAPIToken()
	if !generated || len(tok) != 64 {
		t.Fatalf("upgraded config: generated=%v token=%q, want a fresh token", generated, tok)
	}
}

func TestEnsureAPITokenExplicitlyCleared(t *testing.T) {
	setTestHome(t)
	// A token was generated here (marker written), then the user cleared it.
	tok, generated := EnsureAPIToken()
	if !generated || len(tok) != 64 {
		t.Fatalf("first call: generated=%v token=%q", generated, tok)
	}
	cfg := LoadConfig()
	cfg.APIToken = ""
	SaveConfig(cfg)
	// The cleared token must stick — auth disabled, no regeneration.
	tok2, generated2 := EnsureAPIToken()
	if generated2 || tok2 != "" {
		t.Fatalf("cleared token: generated=%v token=%q, want auth disabled (empty, not regenerated)", generated2, tok2)
	}
	if LoadConfig().APIToken != "" {
		t.Fatal("cleared token was re-populated in config")
	}
}

func TestValidChatID(t *testing.T) {
	cases := []struct {
		id   string
		want bool
	}{
		{"abc", true},
		{"a", true},
		{"ABC123_-", true},
		{strings.Repeat("a", 64), true},
		{strings.Repeat("a", 65), false},
		{"", false},
		{"-abc", false},
		{".abc", false},
		{"a b", false},
		{"a/b", false},
		{"a\\b", false},
		{"a..b", false},
		{"../etc", false},
	}
	for _, c := range cases {
		if got := ValidChatID(c.id); got != c.want {
			t.Errorf("ValidChatID(%q) = %v, want %v", c.id, got, c.want)
		}
	}
}
