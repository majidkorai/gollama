package model

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// resetScanClock clears the models-dir scan throttle so each test starts fresh.
func resetScanClock(t *testing.T) {
	t.Helper()
	scanMu.Lock()
	lastScanTime = time.Time{}
	scanMu.Unlock()
}

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
	setTestHome(t)
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
	// P5-T2: the scan path strips the quantization tag BEFORE normalizing
	// underscores, so the short name is clean.
	if info.Name != "My-Model-Q4-K-M" {
		t.Errorf("Name = %q, want My-Model-Q4-K-M", info.Name)
	}
	if info.ShortName != "my-model" {
		t.Errorf("ShortName = %q, want my-model (quant suffix stripped, P5-T2)", info.ShortName)
	}
}

func TestScanModelsIdempotent(t *testing.T) {
	setTestHome(t)
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
	setTestHome(t)
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
	setTestHome(t)
	writeModelFile(t, "Big-Model-Q4_K_M-00002-of-00004.gguf", 10)
	ScanModels()
	if idx := LoadIndex(); len(idx) != 0 {
		t.Fatalf("index = %v, want empty (no part 1 present)", idx)
	}
}

func TestListModelsSkipsMissingBlobs(t *testing.T) {
	setTestHome(t)
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
	setTestHome(t)
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

// TestResolveModelBlobNotFoundIsSentinel (P3-T4): a missing model returns an
// error wrapping ErrNotFound so handlers can errors.Is it instead of
// string-matching.
func TestResolveModelBlobNotFoundIsSentinel(t *testing.T) {
	setTestHome(t)
	SaveIndex(map[string]ModelInfo{
		"alpha": {Name: "alpha", ShortName: "alpha", BlobPath: writeModelFile(t, "alpha.gguf", 10)},
	})
	if _, err := ResolveModelBlob("nope-1b"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("ResolveModelBlob(missing) err = %v, want errors.Is(ErrNotFound)", err)
	}
}

// TestResolveModelBlobFuzzyDeterministic (P2-T5): when several models match
// a fuzzy query, the winner is deterministic — short-name candidates beat
// substring candidates, and ties within a tier go to the lexicographically
// first index name (the old code returned whichever candidate map iteration
// hit first).
func TestResolveModelBlobFuzzyDeterministic(t *testing.T) {
	setTestHome(t)
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
	if got != pA {
		t.Fatalf("fuzzy resolve = %q, want %q (lexicographic first among substring candidates)", got, pA)
	}

	// A short-name candidate wins over a substring candidate even when the
	// substring candidate's name sorts first.
	pC := writeModelFile(t, "zeta.gguf", 10)
	pD := writeModelFile(t, "alpha-qwen-99b.gguf", 10)
	SaveIndex(map[string]ModelInfo{
		"zeta":  {Name: "zeta", ShortName: "qwen", BlobPath: pC},
		"alpha": {Name: "alpha-qwen-99b-gguf", ShortName: "alpha", BlobPath: pD},
	})
	got, err = ResolveModelBlob("qwen")
	if err != nil {
		t.Fatalf("fuzzy resolve failed: %v", err)
	}
	if got != pC {
		t.Fatalf("fuzzy resolve = %q, want %q (short-name tier beats substring tier)", got, pC)
	}
}

// TestScanSplitModelKeepsExistingKey (v4.3.1): when an entry under a
// different key already points at a split set's part-1 file (a pull indexed
// it first, or an older gollama version named it differently), the scan
// refreshes that entry in place instead of adding a duplicate key — the same
// model must never appear twice in the list.
func TestScanSplitModelKeepsExistingKey(t *testing.T) {
	setTestHome(t)
	writeModelFile(t, "Deep-Seek-0731-UD-IQ4_XS-00001-of-00002.gguf", 100)
	writeModelFile(t, "Deep-Seek-0731-UD-IQ4_XS-00002-of-00002.gguf", 200)
	part1 := filepath.Join(ModelsDir(), "Deep-Seek-0731-UD-IQ4_XS-00001-of-00002.gguf")
	SaveIndex(map[string]ModelInfo{
		"deep-seek-0731-ud-iq4-xs": {Name: "deep-seek-0731-ud-iq4-xs", ShortName: "deep-seek-0731-ud-iq4-xs", BlobPath: part1, Size: 100},
	})
	ScanModels()

	idx := LoadIndex()
	if len(idx) != 1 {
		t.Fatalf("index = %v, want exactly one entry (no duplicate key)", idx)
	}
	info, ok := idx["deep-seek-0731-ud-iq4-xs"]
	if !ok {
		t.Fatalf("pre-existing key should be kept: %v", idx)
	}
	if info.Size != 300 {
		t.Errorf("Size = %d, want 300 (sum of all parts)", info.Size)
	}
}

// TestScanDedupsDuplicateBlobKeys (v4.3.1): a corrupt index holding two keys
// for the same blob file is repaired on the next scan. The larger size
// survives (a part-1-only size loses to the split total); ties go to the
// lexicographically first key.
func TestScanDedupsDuplicateBlobKeys(t *testing.T) {
	setTestHome(t)
	writeModelFile(t, "Solo-Q8_0.gguf", 100)
	p := filepath.Join(ModelsDir(), "Solo-Q8_0.gguf")
	SaveIndex(map[string]ModelInfo{
		"big-key":   {Name: "big-key", ShortName: "big-key", BlobPath: p, Size: 100},
		"small-key": {Name: "small-key", ShortName: "small-key", BlobPath: p, Size: 42},
	})
	ScanModels()

	idx := LoadIndex()
	if len(idx) != 1 {
		t.Fatalf("index = %v, want the duplicate removed", idx)
	}
	if _, ok := idx["big-key"]; !ok {
		t.Fatalf("the larger-size entry should survive: %v", idx)
	}
}

func TestScanDedupTieBreaksLexicographic(t *testing.T) {
	setTestHome(t)
	writeModelFile(t, "Twin.gguf", 100)
	p := filepath.Join(ModelsDir(), "Twin.gguf")
	SaveIndex(map[string]ModelInfo{
		"zeta":  {Name: "zeta", ShortName: "zeta", BlobPath: p, Size: 100},
		"alpha": {Name: "alpha", ShortName: "alpha", BlobPath: p, Size: 100},
	})
	ScanModels()

	idx := LoadIndex()
	if len(idx) != 1 {
		t.Fatalf("index = %v, want the duplicate removed", idx)
	}
	if _, ok := idx["alpha"]; !ok {
		t.Fatalf("lexicographically first key should survive: %v", idx)
	}
}

// TestListModelsDedupsDuplicateBlobs (v4.3.1): even before a scan repairs
// the index on disk, the model list never shows the same blob twice. The
// scan is throttled here so the in-memory dedup is what does the job.
func TestListModelsDedupsDuplicateBlobs(t *testing.T) {
	setTestHome(t)
	scanInterval = time.Hour
	defer func() { scanInterval = 0 }()
	scanMu.Lock()
	lastScanTime = time.Now()
	scanMu.Unlock()

	writeModelFile(t, "Dup-Q4_K_M.gguf", 100)
	p := filepath.Join(ModelsDir(), "Dup-Q4_K_M.gguf")
	SaveIndex(map[string]ModelInfo{
		"dup-big":   {Name: "dup-big", ShortName: "dup-big", BlobPath: p, Size: 100},
		"dup-small": {Name: "dup-small", ShortName: "dup-small", BlobPath: p, Size: 10},
	})
	models, err := ListModels()
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	if len(models) != 1 {
		t.Fatalf("ListModels returned %d entries, want 1: %+v", len(models), models)
	}
	if models[0].Name != "dup-big" {
		t.Fatalf("survivor = %q, want dup-big (larger size)", models[0].Name)
	}
}

// TestScanModelsMaybeThrottles (P4-T2): the throttled scan runs at most once
// per interval; ScanModelsForce bypasses the throttle.
func TestScanModelsMaybeThrottles(t *testing.T) {
	setTestHome(t)
	// Enable throttling for this test only.
	scanInterval = time.Hour
	defer func() { scanInterval = 0 }()
	resetScanClock(t)

	writeModelFile(t, "first.gguf", 10)
	scanModelsMaybe()
	if _, ok := LoadIndex()["first"]; !ok {
		t.Fatalf("first scan should index the file: %v", LoadIndex())
	}

	// Second scan within the interval is throttled — new file not indexed.
	writeModelFile(t, "second.gguf", 10)
	scanModelsMaybe()
	if _, ok := LoadIndex()["second"]; ok {
		t.Fatalf("throttled scan should not index the new file")
	}

	// Force scan bypasses the throttle and indexes the new file.
	ScanModelsForce()
	if _, ok := LoadIndex()["second"]; !ok {
		t.Fatalf("force scan should index the new file: %v", LoadIndex())
	}
}
