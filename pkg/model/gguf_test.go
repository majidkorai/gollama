package model

import (
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
		case 5: // int32
			buf = append(buf, i32le(val.(int32))...)
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
