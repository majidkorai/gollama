package llama

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExpectedChecksum(t *testing.T) {
	content := "aa11bb22cc33dd44ee55ff667788990011223344556677889900aabbccddeeff  gollama-linux-amd64\n" +
		"11223344556677889900aabbccddeeffaa11bb22cc33dd44ee55ff6677889900  gollama-linux-arm64\n" +
		"deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef  gollama-darwin-arm64\n"
	if got := expectedChecksum(content, "gollama-linux-amd64"); got != "aa11bb22cc33dd44ee55ff667788990011223344556677889900aabbccddeeff" {
		t.Errorf("expectedChecksum = %q", got)
	}
	// Binary-mode lines (sha256sum -b) use a leading *.
	binMode := "cafebabe  *gollama-windows-amd64.exe\n"
	if got := expectedChecksum(binMode, "gollama-windows-amd64.exe"); got != "cafebabe" {
		t.Errorf("binary-mode expectedChecksum = %q", got)
	}
	// Unlisted asset → empty (caller warns and skips verification).
	if got := expectedChecksum(content, "gollama-nope"); got != "" {
		t.Errorf("unlisted asset = %q, want empty", got)
	}
}

// newLocalChecksumServer serves a fixed checksums.txt at /<tag>/checksums.txt.
func newLocalChecksumServer(t *testing.T, tag, content string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/"+tag+"/checksums.txt" {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte(content))
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

// TestVerifyChecksumMismatch runs the real verifyChecksum against a local
// server serving a checksums.txt that does NOT match the binary — the update
// must be aborted with a mismatch error.
func TestVerifyChecksumMismatch(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // os.UserHomeDir reads USERPROFILE on Windows

	binPath := filepath.Join(home, "gollama-linux-amd64")
	if err := os.WriteFile(binPath, []byte("gollama-binary-bytes"), 0644); err != nil {
		t.Fatal(err)
	}
	wrongHash := "0000000000000000000000000000000000000000000000000000000000000000"
	checksums := wrongHash + "  gollama-linux-amd64\n"

	old := checksumURLBase
	checksumURLBase = newLocalChecksumServer(t, "v9.9.9", checksums)
	defer func() { checksumURLBase = old }()

	err := verifyChecksum("v9.9.9", "gollama-linux-amd64", binPath)
	if err == nil {
		t.Fatal("expected mismatch error, got nil")
	}
	if !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("error = %q, want a checksum mismatch", err.Error())
	}
}

// TestVerifyChecksumMatch verifies a good checksum passes cleanly.
func TestVerifyChecksumMatch(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // os.UserHomeDir reads USERPROFILE on Windows
	binPath := filepath.Join(home, "gollama-linux-amd64")
	payload := []byte("gollama-binary-bytes")
	if err := os.WriteFile(binPath, payload, 0644); err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(payload)
	checksums := hex.EncodeToString(h[:]) + "  gollama-linux-amd64\n"

	old := checksumURLBase
	checksumURLBase = newLocalChecksumServer(t, "v9.9.9", checksums)
	defer func() { checksumURLBase = old }()

	if err := verifyChecksum("v9.9.9", "gollama-linux-amd64", binPath); err != nil {
		t.Fatalf("matching checksum should pass, got: %v", err)
	}
}

// TestVerifyChecksumMissingFile: no checksums.txt (older releases) → warning,
// not a failure.
func TestVerifyChecksumMissingFile(t *testing.T) {
	home := t.TempDir()
	binPath := filepath.Join(home, "gollama-linux-amd64")
	if err := os.WriteFile(binPath, []byte("bytes"), 0644); err != nil {
		t.Fatal(err)
	}
	old := checksumURLBase
	checksumURLBase = newLocalChecksumServer(t, "v9.9.9", "whatever")
	defer func() { checksumURLBase = old }()

	// Ask for a different tag than the server knows → 404.
	if err := verifyChecksum("v8.0.0", "gollama-linux-amd64", binPath); err != nil {
		t.Fatalf("missing checksums.txt should warn, not fail: %v", err)
	}
}

func TestCompareBuildNumbers(t *testing.T) {
	cases := []struct {
		name       string
		installed  string
		latest     string
		wantBehind int
		wantComp   bool
	}{
		{"equal", "b500", "b500", 0, true},
		{"behind", "b500", "b999", 499, true},
		{"ahead", "b999", "b500", 0, true},
		{"b-prefixed both", "b123", "b456", 333, true},
		{"bare numbers", "300", "350", 50, true},
		{"version output format", "version 396 (b396)", "b400", 4, true},
		{"unparseable installed", "custom-build", "b500", 0, false},
		{"unparseable latest", "b500", "unknown", 0, false},
		{"both unparseable", "custom", "unknown", 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			behind, comparable := CompareBuildNumbers(tc.installed, tc.latest)
			if behind != tc.wantBehind || comparable != tc.wantComp {
				t.Errorf("CompareBuildNumbers(%q, %q) = (%d, %v), want (%d, %v)",
					tc.installed, tc.latest, behind, comparable, tc.wantBehind, tc.wantComp)
			}
		})
	}
}

// TestLatestReleaseInfoTTL verifies the 1h TTL cache: the second call within
// the TTL is served from cache (no second upstream fetch).
func TestLatestReleaseInfoTTL(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `[{"tag_name":"v0.2.0","html_url":"https://example.com/v0.2.0","assets":[]},`+
			`{"tag_name":"b500","html_url":"https://example.com/b500","assets":[]}]`)
	}))
	defer srv.Close()
	old := releaseAPIBase
	releaseAPIBase = srv.URL
	defer func() { releaseAPIBase = old }()
	os.Unsetenv("GOLLAMA_RELEASE_API_BASE") // ensure the package var is used

	tag, url, err := LatestReleaseInfo()
	if err != nil {
		t.Fatalf("LatestReleaseInfo: %v", err)
	}
	if tag != "b500" || url != "https://example.com/b500" {
		t.Errorf("LatestReleaseInfo = (%q, %q), want (b500, https://example.com/b500)", tag, url)
	}
	if hits != 1 {
		t.Fatalf("first call hits = %d, want 1", hits)
	}
	if _, _, err := LatestReleaseInfo(); err != nil {
		t.Fatalf("second LatestReleaseInfo: %v", err)
	}
	if hits != 1 {
		t.Errorf("second call re-fetched: hits = %d, want 1 (served from cache)", hits)
	}
}
