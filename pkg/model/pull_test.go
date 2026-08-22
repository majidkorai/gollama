package model

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeHF emulates the HuggingFace API + CDN endpoints, honoring Range
// requests like the real CDN does.
type fakeHF struct {
	mu sync.Mutex

	files map[string][]byte // "modelID/filename" -> content

	// apiSizes optionally overrides the size the API metadata reports for a
	// file (to simulate metadata lying about the size).
	apiSizes map[string]int64

	// slow makes resolve responses drip in 4 KiB chunks with 10 ms pauses so
	// tests can observe an in-flight download.
	slow bool

	// done is closed by the test to unblock slow handlers.
	done chan struct{}

	resolves int      // GET /resolve/main/ requests
	ranges   []string // Range headers seen on resolve GETs
}

func (f *fakeHF) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/models/", func(w http.ResponseWriter, r *http.Request) {
		modelID := strings.TrimPrefix(r.URL.Path, "/api/models/")
		type sib struct {
			Filename string `json:"rfilename"`
			Size     int64  `json:"size"`
		}
		resp := struct {
			Siblings []sib `json:"siblings"`
		}{}
		for name, content := range f.files {
			if strings.HasPrefix(name, modelID+"/") {
				size := int64(len(content))
				if s, ok := f.apiSizes[name]; ok {
					size = s
				}
				resp.Siblings = append(resp.Siblings, sib{Filename: name, Size: size})
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		sep := strings.Index(path, "/resolve/main/")
		if sep < 0 {
			http.NotFound(w, r)
			return
		}
		// The URL embeds the full repo path in the filename segment, like the
		// real CDN: {hfBase}/{modelID}/resolve/main/{modelID}/{file...}
		filename := path[sep+len("/resolve/main/"):]
		content, ok := f.files[filename]
		if !ok {
			http.NotFound(w, r)
			return
		}

		isGet := r.Method == http.MethodGet
		f.mu.Lock()
		if isGet {
			f.resolves++
			f.ranges = append(f.ranges, r.Header.Get("Range"))
		}
		slow := f.slow
		f.mu.Unlock()

		start, end := int64(0), int64(len(content))-1
		rangeHeader := r.Header.Get("Range")
		if isGet && rangeHeader != "" {
			// Expect "bytes=N-"
			lo, _, _ := strings.Cut(strings.TrimPrefix(rangeHeader, "bytes="), "-")
			if n, err := strconv.ParseInt(lo, 10, 64); err == nil {
				start = n
			}
		}
		if start >= int64(len(content)) {
			w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
			return
		}

		if slow {
			buf := make([]byte, 4096)
			for i := start; i < end+1; i += int64(len(buf)) {
				n := copy(buf, content[i:min64(i+int64(len(buf)), int64(len(content)))] )
				w.Write(buf[:n])
				if fl, ok := w.(http.Flusher); ok {
					fl.Flush()
				}
				select {
				case <-f.done:
					return
				case <-time.After(10 * time.Millisecond):
				}
			}
			return
		}

		if isGet && rangeHeader != "" {
			w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(content)))
			w.WriteHeader(http.StatusPartialContent)
			w.Write(content[start : end+1])
			return
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(content)))
		w.Write(content)
	})
	return mux
}

func min64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

// setTestHF starts a fake HF server and points hfBaseURL at it.
func setTestHF(t *testing.T, files map[string][]byte) *fakeHF {
	t.Helper()
	f := &fakeHF{files: files, done: make(chan struct{})}
	srv := httptest.NewServer(f.handler())
	t.Cleanup(srv.Close)
	t.Cleanup(func() { close(f.done) })
	old := hfBaseURL
	hfBaseURL = srv.URL
	t.Cleanup(func() { hfBaseURL = old })
	return f
}

func TestPullModelFreshDownload(t *testing.T) {
	setTestHome(t)
	content := []byte(strings.Repeat("gollama-test-data-", 100))
	f := setTestHF(t, map[string][]byte{
		"test/repo/test-repo-Q4_K_M.gguf": content,
	})

	if err := PullModel("test/repo:Q4_K_M"); err != nil {
		t.Fatalf("PullModel: %v", err)
	}

	dest := filepath.Join(ModelsDir(), "test-repo-Q4_K_M.gguf")
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("reading dest: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Fatalf("content mismatch: got %d bytes, want %d", len(got), len(content))
	}
	if _, err := os.Stat(dest + ".part"); !os.IsNotExist(err) {
		t.Fatalf("leftover .part file should not exist")
	}
	if f.resolves != 1 {
		t.Fatalf("resolve requests = %d, want 1", f.resolves)
	}
}

// TestPullModelQuantMatchCaseInsensitive guards the HF quant-suffix match:
// repos commonly use lowercase filenames (q4_k_m.gguf) while users request the
// canonical uppercase form (Q4_K_M). A case-sensitive match would fail and
// fall back to downloading *all* GGUF files (e.g. a huge fp16 build).
func TestPullModelQuantMatchCaseInsensitive(t *testing.T) {
	setTestHome(t)
	q4 := []byte(strings.Repeat("q4-lowercase-content-", 100))
	fp16 := []byte(strings.Repeat("fp16-large-content--", 1000))
	f := setTestHF(t, map[string][]byte{
		"test/repo/test-repo-q4_k_m.gguf": q4,
		"test/repo/test-repo-fp16.gguf":   fp16,
	})

	if err := PullModel("test/repo:Q4_K_M"); err != nil {
		t.Fatalf("PullModel: %v", err)
	}

	// Only the q4_k_m file should be downloaded — not the fp16 fallback.
	if f.resolves != 1 {
		t.Fatalf("resolve requests = %d, want 1 (case-insensitive quant match)", f.resolves)
	}
	dest := filepath.Join(ModelsDir(), "test-repo-q4_k_m.gguf")
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("reading dest: %v", err)
	}
	if !bytes.Equal(got, q4) {
		t.Fatalf("content mismatch: got %d bytes, want %d", len(got), len(q4))
	}
	if _, err := os.Stat(filepath.Join(ModelsDir(), "test-repo-fp16.gguf")); !os.IsNotExist(err) {
		t.Fatalf("fp16 file should not have been downloaded (quant match failed)")
	}
}

func TestPullModelResumesPartial(t *testing.T) {
	setTestHome(t)
	content := []byte(strings.Repeat("resume-resume-resume-", 100))
	f := setTestHF(t, map[string][]byte{
		"test/repo/test-repo-Q4_K_M.gguf": content,
	})
	dest := filepath.Join(ModelsDir(), "test-repo-Q4_K_M.gguf")
	EnsureDir(ModelsDir())
	part := dest + ".part"
	half := len(content) / 2
	if err := os.WriteFile(part, content[:half], 0644); err != nil {
		t.Fatal(err)
	}

	if err := PullModel("test/repo:Q4_K_M"); err != nil {
		t.Fatalf("PullModel: %v", err)
	}

	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("reading dest: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Fatalf("resumed content mismatch: got %d bytes, want %d", len(got), len(content))
	}
	if _, err := os.Stat(part); !os.IsNotExist(err) {
		t.Fatalf("leftover .part file should not exist")
	}

	var sawRange bool
	for _, r := range f.ranges {
		if r == fmt.Sprintf("bytes=%d-", half) {
			sawRange = true
		}
	}
	if !sawRange {
		t.Fatalf("expected Range: bytes=%d- among %v", half, f.ranges)
	}
}

func TestPullModelStaleOversizedPartRestarted(t *testing.T) {
	setTestHome(t)
	content := []byte(strings.Repeat("stale-part-content-", 100))
	setTestHF(t, map[string][]byte{
		"test/repo/test-repo-Q4_K_M.gguf": content,
	})
	dest := filepath.Join(ModelsDir(), "test-repo-Q4_K_M.gguf")
	EnsureDir(ModelsDir())
	part := dest + ".part"
	// A partial file larger than the remote file: stale from some other download.
	if err := os.WriteFile(part, append(append([]byte{}, content...), []byte("stale-extra")...), 0644); err != nil {
		t.Fatal(err)
	}

	if err := PullModel("test/repo:Q4_K_M"); err != nil {
		t.Fatalf("PullModel: %v", err)
	}

	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("reading dest: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Fatalf("content mismatch: got %d bytes, want %d", len(got), len(content))
	}
	if _, err := os.Stat(part); !os.IsNotExist(err) {
		t.Fatalf("leftover .part file should not exist")
	}
}

func TestPullModelCompletePartFinalizedWithoutDownload(t *testing.T) {
	setTestHome(t)
	content := []byte(strings.Repeat("complete-part-data-", 100))
	f := setTestHF(t, map[string][]byte{
		"test/repo/test-repo-Q4_K_M.gguf": content,
	})
	dest := filepath.Join(ModelsDir(), "test-repo-Q4_K_M.gguf")
	EnsureDir(ModelsDir())
	part := dest + ".part"
	// A previous run finished the bytes but died before the rename.
	if err := os.WriteFile(part, content, 0644); err != nil {
		t.Fatal(err)
	}

	if err := PullModel("test/repo:Q4_K_M"); err != nil {
		t.Fatalf("PullModel: %v", err)
	}

	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("reading dest: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Fatalf("content mismatch: got %d bytes, want %d", len(got), len(content))
	}
	if f.resolves != 0 {
		t.Fatalf("resolve requests = %d, want 0 (should have just renamed)", f.resolves)
	}
}

func TestPullModelSizeMismatchDeletesPart(t *testing.T) {
	setTestHome(t)
	content := []byte(strings.Repeat("mismatch-test-data-", 100))
	f := setTestHF(t, map[string][]byte{
		"test/repo/test-repo-Q4_K_M.gguf": content,
	})
	// The API metadata claims the file is bigger than it really is.
	f.apiSizes = map[string]int64{
		"test/repo/test-repo-Q4_K_M.gguf": int64(len(content)) + 100,
	}

	err := PullModel("test/repo:Q4_K_M")
	if err == nil {
		t.Fatal("expected a size mismatch error")
	}
	if !strings.Contains(err.Error(), "size mismatch") {
		t.Fatalf("error = %v, want size mismatch", err)
	}
	dest := filepath.Join(ModelsDir(), "test-repo-Q4_K_M.gguf")
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatalf("dest should not exist after size mismatch")
	}
	if _, err := os.Stat(dest + ".part"); !os.IsNotExist(err) {
		t.Fatalf(".part should be deleted after size mismatch")
	}
}

func TestPullModelCancelDeletesPart(t *testing.T) {
	setTestHome(t)
	content := make([]byte, 1<<20)
	for i := range content {
		content[i] = byte(i % 251)
	}
	f := setTestHF(t, map[string][]byte{
		"test/repo/test-repo-Q4_K_M.gguf": content,
	})
	f.slow = true

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- PullModelWithCallbackContext(ctx, "test/repo:Q4_K_M", nil)
	}()

	part := filepath.Join(ModelsDir(), "test-repo-Q4_K_M.gguf.part")
	deadline := time.Now().Add(10 * time.Second)
	for {
		if fi, err := os.Stat(part); err == nil && fi.Size() > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("partial file never appeared")
		}
		time.Sleep(5 * time.Millisecond)
	}

	cancel()
	err := <-errCh
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if _, err := os.Stat(part); !os.IsNotExist(err) {
		t.Fatal("partial file should be removed on cancel")
	}
}

func TestScanModelsSkipsPartAndZeroByte(t *testing.T) {
	setTestHome(t)
	dir := ModelsDir()
	EnsureDir(dir)
	if err := os.WriteFile(filepath.Join(dir, "real-model.gguf"), []byte("not a real gguf, just a name"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "in-progress.gguf.part"), []byte("partial"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "empty.gguf"), nil, 0644); err != nil {
		t.Fatal(err)
	}

	ScanModels()

	idx := unsafeLoadIndex()
	if len(idx) != 1 {
		t.Fatalf("index has %d entries, want 1: %v", len(idx), idx)
	}
	for name, info := range idx {
		if !strings.HasSuffix(info.BlobPath, "real-model.gguf") {
			t.Fatalf("unexpected indexed entry %q -> %s", name, info.BlobPath)
		}
	}
}
