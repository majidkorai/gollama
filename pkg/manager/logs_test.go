package manager

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTailLogFile(t *testing.T) {
	dir := t.TempDir()

	// Small file: returned in full.
	smallPath := filepath.Join(dir, "small.log")
	if err := os.WriteFile(smallPath, []byte("a\nb\nc\n"), 0644); err != nil {
		t.Fatal(err)
	}
	got, err := TailLogFile(smallPath, 1024)
	if err != nil {
		t.Fatalf("TailLogFile(small): %v", err)
	}
	if string(got) != "a\nb\nc\n" {
		t.Fatalf("small file = %q, want full content", got)
	}

	// Large file: a suffix, <= maxBytes, starting on a line boundary.
	var buf strings.Builder
	for i := 0; i < 500; i++ {
		fmt.Fprintf(&buf, "line-%03d ", i)
		buf.WriteString(strings.Repeat("x", 40))
		buf.WriteByte('\n')
	}
	content := buf.String()
	largePath := filepath.Join(dir, "large.log")
	if err := os.WriteFile(largePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	const maxBytes = 1000
	got, err = TailLogFile(largePath, maxBytes)
	if err != nil {
		t.Fatalf("TailLogFile(large): %v", err)
	}
	gs := string(got)
	if !strings.HasSuffix(content, gs) {
		t.Fatal("tail is not a suffix of the content")
	}
	if len(gs) > int(maxBytes) {
		t.Fatalf("tail = %d bytes, want <= %d", len(gs), maxBytes)
	}
	if offset := len(content) - len(gs); offset > 0 && content[offset-1] != '\n' {
		t.Fatalf("tail does not start on a line boundary (preceding byte = %q)", content[offset-1])
	}
}

func TestRotatingLogWriterRotates(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "port-8080.log")
	const maxBytes = 100

	w, err := newRotatingLogWriter(path, maxBytes)
	if err != nil {
		t.Fatal(err)
	}

	chunk := strings.Repeat("y", 60)
	for i := 0; i < 5; i++ {
		if _, err := w.Write([]byte(chunk)); err != nil {
			t.Fatal(err)
		}
	}
	w.mu.Lock()
	w.file.Close()
	w.mu.Unlock()

	if _, err := os.Stat(path + ".1"); err != nil {
		t.Fatalf("rotated file missing after exceeding maxBytes: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("active log missing after rotation: %v", err)
	}
}

func TestPrepareInstanceLogMax(t *testing.T) {
	dir := t.TempDir()

	// Over maxBytes → rotated to .1, original moved away.
	largePath := filepath.Join(dir, "port-1.log")
	if err := os.WriteFile(largePath, []byte(strings.Repeat("z", 200)), 0644); err != nil {
		t.Fatal(err)
	}
	prepareInstanceLogMax(largePath, 100)
	if _, err := os.Stat(largePath + ".1"); err != nil {
		t.Fatalf("large log not rotated to .1: %v", err)
	}
	if _, err := os.Stat(largePath); !os.IsNotExist(err) {
		t.Fatal("large log should be moved to .1, not left in place")
	}

	// Under maxBytes → deleted (fresh log, old truncate-on-start behavior).
	smallPath := filepath.Join(dir, "port-2.log")
	if err := os.WriteFile(smallPath, []byte("small"), 0644); err != nil {
		t.Fatal(err)
	}
	prepareInstanceLogMax(smallPath, 100)
	if _, err := os.Stat(smallPath); !os.IsNotExist(err) {
		t.Fatal("small log should be deleted")
	}

	// A pre-existing .1 is overwritten on the next rotation.
	againPath := filepath.Join(dir, "port-3.log")
	if err := os.WriteFile(againPath+".1", []byte("old history"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(againPath, []byte(strings.Repeat("q", 200)), 0644); err != nil {
		t.Fatal(err)
	}
	prepareInstanceLogMax(againPath, 100)
	data, err := os.ReadFile(againPath + ".1")
	if err != nil {
		t.Fatalf(".1 missing: %v", err)
	}
	if string(data) != strings.Repeat("q", 200) {
		t.Fatalf(".1 not overwritten: %q", data)
	}
}
