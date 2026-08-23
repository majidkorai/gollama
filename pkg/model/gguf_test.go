package model

import (
	"math"
	"os"
	"testing"
)

func buildGGUF(t *testing.T, kvs ...string) string {
	t.Helper()
	f, err := os.CreateTemp("", "gguf-test-*.gguf")
	if err != nil {
		t.Fatal(err)
	}
	path := f.Name()

	// Count metadata pairs
	metaCount := len(kvs) / 2

	buf := []byte("GGUF")
	buf = append(buf, u32le(3)...)        // version
	buf = append(buf, u64le(0)...)        // tensor count
	buf = append(buf, u64le(uint64(metaCount))...) // metadata count

	for i := 0; i < len(kvs); i += 2 {
		key := kvs[i]
		val := kvs[i+1]
		buf = append(buf, ggufString(key)...)
		// Default: treat as string (type 8)
		buf = append(buf, u32le(8)...)
		buf = append(buf, ggufString(val)...)
	}

	if _, err := f.Write(buf); err != nil {
		f.Close()
		os.Remove(path)
		t.Fatal(err)
	}
	f.Close()
	return path
}

func buildGGUFWithTypes(t *testing.T, entries ...interface{}) string {
	t.Helper()
	f, err := os.CreateTemp("", "gguf-test-*.gguf")
	if err != nil {
		t.Fatal(err)
	}
	path := f.Name()

	metaCount := uint64(0)
	var buf []byte

	// First pass: count
	for i := 0; i < len(entries); i += 3 {
		metaCount++
	}

	buf = append(buf, []byte("GGUF")...)
	buf = append(buf, u32le(3)...)
	buf = append(buf, u64le(0)...)
	buf = append(buf, u64le(metaCount)...)

	for i := 0; i < len(entries); i += 3 {
		key := entries[i].(string)
		typ := entries[i+1].(uint32)
		val := entries[i+2]
		buf = append(buf, ggufString(key)...)
		buf = append(buf, u32le(typ)...)
		switch typ {
		case 4: // uint32
			buf = append(buf, u32le(val.(uint32))...)
		case 5: // int32
			buf = append(buf, i32le(val.(int32))...)
		case 6: // float32
			bits := math.Float32bits(val.(float32))
			buf = append(buf, u32le(bits)...)
		case 8: // string
			buf = append(buf, ggufString(val.(string))...)
		case 10: // uint64
			buf = append(buf, u64le(val.(uint64))...)
		default:
			t.Fatalf("unsupported type %d in test builder", typ)
		}
	}

	if _, err := f.Write(buf); err != nil {
		f.Close()
		os.Remove(path)
		t.Fatal(err)
	}
	f.Close()
	return path
}

func u32le(v uint32) []byte {
	return []byte{byte(v), byte(v >> 8), byte(v >> 16), byte(v >> 24)}
}
func u64le(v uint64) []byte {
	return []byte{byte(v), byte(v >> 8), byte(v >> 16), byte(v >> 24), byte(v >> 32), byte(v >> 40), byte(v >> 48), byte(v >> 56)}
}
func i32le(v int32) []byte {
	return u32le(uint32(v))
}
func ggufString(s string) []byte {
	return append(u64le(uint64(len(s))), []byte(s)...)
}

func TestReadGGUFMetadata_Architecture(t *testing.T) {
	path := buildGGUFWithTypes(t,
		"general.architecture", uint32(8), "llama",
		"general.file_type", uint32(5), int32(12), // Q4_K
	)
	defer os.Remove(path)

	meta, err := readGGUFMetadata(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta == nil {
		t.Fatal("expected metadata, got nil")
	}
	if meta.Architecture != "llama" {
		t.Errorf("expected architecture=llama, got %q", meta.Architecture)
	}
	if meta.Quantization != "Q4_K" {
		t.Errorf("expected quantization=Q4_K, got %q", meta.Quantization)
	}
}

func TestReadGGUFMetadata_ContextLength(t *testing.T) {
	path := buildGGUFWithTypes(t,
		"general.architecture", uint32(8), "gemma",
		"llama.context_length", uint32(10), uint64(8192),
		"general.file_type", uint32(5), int32(2),
	)
	defer os.Remove(path)

	meta, err := readGGUFMetadata(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta == nil {
		t.Fatal("expected metadata, got nil")
	}
	if meta.Architecture != "gemma" {
		t.Errorf("expected architecture=gemma, got %q", meta.Architecture)
	}
	if meta.Quantization != "Q4_0" {
		t.Errorf("expected quantization=Q4_0, got %q", meta.Quantization)
	}
	if meta.ContextLength != 8192 {
		t.Errorf("expected context_length=8192, got %d", meta.ContextLength)
	}
}

func TestReadGGUFMetadata_UnknownFileType(t *testing.T) {
	path := buildGGUFWithTypes(t,
		"general.architecture", uint32(8), "test",
		"general.file_type", uint32(5), int32(999),
	)
	defer os.Remove(path)

	meta, err := readGGUFMetadata(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta == nil {
		t.Fatal("expected metadata, got nil")
	}
	if meta.Quantization != "type_999" {
		t.Errorf("expected quantization=type_999, got %q", meta.Quantization)
	}
}

// TestReadGGUFMetadata_Float32DoesNotDesync (v4.3.1): a FLOAT32 metadata
// value (type 6, 4 bytes) used to be skipped as 8 bytes, desynchronizing
// the stream and failing the whole parse — so architecture/quant/context
// went missing on every file with general.sampling.* or other float keys
// (common in recent quant repos).
func TestReadGGUFMetadata_Float32DoesNotDesync(t *testing.T) {
	path := buildGGUFWithTypes(t,
		"general.architecture", uint32(8), "deepseek4",
		"general.sampling.top_p", uint32(6), float32(0.95),
		"general.sampling.temp", uint32(6), float32(1.0),
		"general.name", uint32(8), "Test-Model-0731",
		"general.file_type", uint32(5), int32(23), // IQ4_XS
	)
	defer os.Remove(path)

	meta, err := readGGUFMetadata(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta == nil {
		t.Fatal("expected metadata, got nil")
	}
	if meta.Architecture != "deepseek4" {
		t.Errorf("architecture = %q, want deepseek4 (parse desynced?)", meta.Architecture)
	}
	if meta.Quantization != "IQ4_XS" {
		t.Errorf("quantization = %q, want IQ4_XS (parse desynced?)", meta.Quantization)
	}
}

// TestReadGGUFMetadata_UInt32ContextAndBlocks (v4.3.1): recent writers
// store <arch>.context_length and <arch>.block_count as UINT32 (type 4);
// the parser only understood UINT64/INT32, so the ctx badge went missing.
func TestReadGGUFMetadata_UInt32ContextAndBlocks(t *testing.T) {
	path := buildGGUFWithTypes(t,
		"general.architecture", uint32(8), "qwen35",
		"qwen35.context_length", uint32(4), uint32(262144),
		"qwen35.block_count", uint32(4), uint32(65),
	)
	defer os.Remove(path)

	meta, err := readGGUFMetadata(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta == nil {
		t.Fatal("expected metadata, got nil")
	}
	if meta.ContextLength != 262144 {
		t.Errorf("context_length = %d, want 262144", meta.ContextLength)
	}
	if meta.BlockCount != 65 {
		t.Errorf("block_count = %d, want 65", meta.BlockCount)
	}
}

func TestReadGGUFMetadata_NotGGUF(t *testing.T) {
	f, err := os.CreateTemp("", "not-gguf-*.bin")
	if err != nil {
		t.Fatal(err)
	}
	path := f.Name()
	f.Write([]byte("not a gguf file"))
	f.Close()
	defer os.Remove(path)

	meta, err := readGGUFMetadata(path)
	if err != nil {
		t.Fatalf("expected no error for non-GGUF, got: %v", err)
	}
	if meta != nil {
		t.Fatalf("expected nil for non-GGUF, got %+v", meta)
	}
}

func TestReadGGUFMetadata_Empty(t *testing.T) {
	f, err := os.CreateTemp("", "empty-*.gguf")
	if err != nil {
		t.Fatal(err)
	}
	path := f.Name()
	f.Close()
	defer os.Remove(path)

	_, err = readGGUFMetadata(path)
	if err == nil {
		t.Fatal("expected error for empty file")
	}
}

func TestStripQuantSuffix(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		// Standard K-quants
		{"qwen2-0.5b-instruct-q4_k_m", "qwen2-0.5b-instruct"},
		{"qwen2-0.5b-instruct-q3_k_s", "qwen2-0.5b-instruct"},
		{"llama-3-8b-instruct-q2_k", "llama-3-8b-instruct"},
		{"llama-3-8b-instruct-q6_k", "llama-3-8b-instruct"},
		{"llama-3-8b-instruct-q8_k", "llama-3-8b-instruct"},
		// Old-style quants
		{"llama-3-8b-instruct-q4_0", "llama-3-8b-instruct"},
		{"llama-3-8b-instruct-q4_1", "llama-3-8b-instruct"},
		{"llama-3-8b-instruct-q5_1", "llama-3-8b-instruct"},
		{"llama-3-8b-instruct-q8_0", "llama-3-8b-instruct"},
		// IQ quants
		{"deepseek-v4-flash-81gb-iq2_xxs", "deepseek-v4-flash-81gb"},
		{"gemma-2-2b-it-iq4_xs", "gemma-2-2b-it"},
		{"gemma-2-2b-it-iq4_nl", "gemma-2-2b-it"},
		{"llama-3-8b-instruct-iq1_s", "llama-3-8b-instruct"},
		{"llama-3-8b-instruct-iq3_xxs", "llama-3-8b-instruct"},
		// Vendor prefixes (UD-, ARM-) — longer matches must win over the bare quant
		{"qwen2-1.5b-instruct-ud-q4_k_m", "qwen2-1.5b-instruct"},
		{"llama-3-8b-instruct-ud-q3_k_l", "llama-3-8b-instruct"},
		{"llama-3-8b-instruct-arm-q4_k_m", "llama-3-8b-instruct"},
		{"llama-3-8b-instruct-arm-q8_0", "llama-3-8b-instruct"},
		// FP formats
		{"gemma-2-2b-it-bf16", "gemma-2-2b-it"},
		{"llama-3-8b-f16", "llama-3-8b"},
		{"llama-3-8b-f32", "llama-3-8b"},
		// With .gguf extension
		{"Qwen2-0.5B-Instruct-Q4_K_M.gguf", "Qwen2-0.5B-Instruct"},
		// Underscore separators
		{"model_q4_k_m", "model"},
		{"model_ud-q4_k_m", "model"},
		// No quant → unchanged (case preserved)
		{"my-model", "my-model"},
		{"gemma-4-12b-it", "gemma-4-12b-it"},
		{"My-Model-7B", "My-Model-7B"},
		// Case-insensitive match, original case preserved
		{"My-Model-Q4_K_M", "My-Model"},
	}
	for _, tc := range cases {
		if got := StripQuantSuffix(tc.in); got != tc.want {
			t.Errorf("StripQuantSuffix(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestGGUFMetadataCached(t *testing.T) {
	path := buildGGUFWithTypes(t,
		"general.architecture", uint32(8), "llama",
		"llama.block_count", uint32(5), int32(31),
	)
	defer os.Remove(path)

	m1, err := GGUFMetadataCached(path)
	if err != nil {
		t.Fatal(err)
	}
	if m1 == nil || m1.BlockCount != 31 {
		t.Fatalf("meta = %+v, want BlockCount 31", m1)
	}
	// Remove the file: a second lookup must still succeed from the cache
	// (proving it is not re-reading the file).
	os.Remove(path)
	m2, err := GGUFMetadataCached(path)
	if err != nil {
		t.Fatal(err)
	}
	if m2 != m1 {
		t.Error("expected the same cached *GGUFMeta pointer on the second lookup")
	}
}
