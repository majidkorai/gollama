package model

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// writeModelFile creates a fake .gguf in the (temp-HOME) models dir. The
// content is not a valid GGUF (magic check fails), so metadata parsing
// returns nil — the index entry stays metadata-free.
func writeModelFile(t *testing.T, name string, size int) string {
	t.Helper()
	p := filepath.Join(ModelsDir(), name)
	EnsureDir(ModelsDir())
	if err := os.WriteFile(p, bytes.Repeat([]byte{0x47}, size), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestScanModelsIndexesNewFiles(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	writeModelFile(t, "My-Model-Q4_K_M.gguf", 100)
	ScanModels()

	idx := LoadIndex()
	info, ok := idx["My-Model-Q4_K_M"]
	if !ok {
		t.Fatalf("index = %v, want key My-Model-Q4_K_M", idx)
	}
	if info.Source != "local" {
		t.Errorf("Source = %q, want local", info.Source)
	}
	if info.Size != 100 {
		t.Errorf("Size = %d, want 100", info.Size)
	}
	// Documents current behavior: the scan path replaces underscores with
	// hyphens in Name BEFORE the quant-stripping regex runs (the regex
	// expects underscores), so the short name keeps the quant suffix.
	if info.Name != "My-Model-Q4-K-M" {
		t.Errorf("Name = %q, want My-Model-Q4-K-M", info.Name)
	}
	if info.ShortName != "my-model-q4-k-m" {
		t.Errorf("ShortName = %q, want my-model-q4-k-m (documents current behavior)", info.ShortName)
	}
}

func TestScanModelsIdempotent(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	writeModelFile(t, "A-B-Q4_K_M.gguf", 10)
	ScanModels()
	first := LoadIndex()
	if len(first) != 1 {
		t.Fatalf("after first scan index has %d entries, want 1", len(first))
	}
	ScanModels()
	second := LoadIndex()
	if len(second) != 1 {
		t.Fatalf("after second scan index has %d entries, want 1 (no duplicates)", len(second))
	}
}

func TestScanModelsSplitFiles(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	for i := 1; i <= 4; i++ {
		writeModelFile(t, fmt.Sprintf("Big-Model-Q4_K_M-%05d-of-00004.gguf", i), 10)
	}
	ScanModels()

	idx := LoadIndex()
	if len(idx) != 1 {
		t.Fatalf("index = %v, want exactly one entry for the split", idx)
	}
	info, ok := idx["big-model"]
	if !ok {
		t.Fatalf("no entry with key big-model: %v", idx)
	}
	if info.Size != 40 {
		t.Errorf("Size = %d, want 40 (sum of all parts)", info.Size)
	}
	if info.Source != "local" {
		t.Errorf("Source = %q, want local", info.Source)
	}
	// BlobPath points at part 1; llama-server discovers the rest by naming
	// convention.
	if filepath.Base(info.BlobPath) != "Big-Model-Q4_K_M-00001-of-00004.gguf" {
		t.Errorf("BlobPath = %q, want part 1", info.BlobPath)
	}
}

func TestScanModelsSplitWithoutFirstPart(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	writeModelFile(t, "Big-Model-Q4_K_M-00002-of-00004.gguf", 10)
	ScanModels()
	if idx := LoadIndex(); len(idx) != 0 {
		t.Fatalf("index = %v, want empty (no part 1 present)", idx)
	}
}

func TestListModelsSkipsMissingBlobs(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	p := writeModelFile(t, "real.gguf", 10)
	SaveIndex(map[string]ModelInfo{
		"real":  {Name: "real", ShortName: "real", BlobPath: p},
		"ghost": {Name: "ghost", ShortName: "ghost", BlobPath: filepath.Join(ModelsDir(), "gone.gguf")},
	})
	models, err := ListModels()
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	if len(models) != 1 || models[0].Name != "real" {
		t.Fatalf("ListModels = %+v, want only the existing blob", models)
	}
}

func TestResolveModelBlob(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	pA := writeModelFile(t, "alpha.gguf", 10)
	pB := writeModelFile(t, "beta.gguf", 10)
	SaveIndex(map[string]ModelInfo{
		"alpha": {Name: "alpha", ShortName: "alpha", BlobPath: pA},
		"beta":  {Name: "beta", ShortName: "beta", BlobPath: pB},
	})

	if got, err := ResolveModelBlob("alpha"); err != nil || got != pA {
		t.Fatalf("exact match = %q, %v", got, err)
	}
	if got, err := ResolveModelBlob(pB); err != nil || got != pB {
		t.Fatalf("direct path = %q, %v", got, err)
	}
	if _, err := ResolveModelBlob("gamma"); err == nil {
		t.Fatal("expected error for missing model")
	}
}

// TestResolveModelBlobFuzzyDocumentsNondeterminism: two models both contain
// the substring "qwen"; the current implementation returns whichever comes
// first in map iteration order, which varies between runs. This test pins
// that it returns a valid candidate (not which one) — P2-T5 makes the choice
// deterministic and this test can then assert the exact winner.
func TestResolveModelBlobFuzzyDocumentsNondeterminism(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	pA := writeModelFile(t, "qwen-a.gguf", 10)
	pB := writeModelFile(t, "qwen-b.gguf", 10)
	SaveIndex(map[string]ModelInfo{
		"qwen-a": {Name: "qwen-a", ShortName: "qwen-a", BlobPath: pA},
		"qwen-b": {Name: "qwen-b", ShortName: "qwen-b", BlobPath: pB},
	})
	got, err := ResolveModelBlob("qwen")
	if err != nil {
		t.Fatalf("fuzzy resolve failed: %v", err)
	}
	if got != pA && got != pB {
		t.Fatalf("fuzzy resolve = %q, want one of the candidates", got)
	}
}
