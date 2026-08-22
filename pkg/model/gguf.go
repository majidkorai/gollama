package model

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

const ggufMagic = "GGUF"

var ggufTypeNames = map[uint32]string{
	0:  "F32",
	1:  "F16",
	2:  "Q4_0",
	3:  "Q4_1",
	6:  "Q5_0",
	7:  "Q5_1",
	8:  "Q8_0",
	9:  "Q8_1",
	10: "Q2_K",
	11: "Q3_K",
	12: "Q4_K",
	13: "Q5_K",
	14: "Q6_K",
	15: "Q8_K",
	16: "IQ2_XXS",
	17: "IQ2_XS",
	18: "IQ3_XXS",
	19: "IQ1_S",
	20: "IQ4_NL",
	21: "IQ3_S",
	22: "IQ2_S",
	23: "IQ2_M",
	24: "IQ4_XS",
	25: "IQ1_M",
	26: "BF16",
}

// GGUFMeta is the subset of GGUF metadata gollama cares about.
type GGUFMeta struct {
	Architecture   string
	Quantization   string
	ContextLength  uint64
	FileType       uint32
	BlockCount     uint32
}

// ggufMetaCache memoizes parsed GGUF metadata by path for the lifetime of
// the process. Model files are immutable in practice (re-pulls of an
// existing file are refused), so caching is safe on the hot path
// (per-instance-start metadata lookups).
var (
	ggufMetaCache   = map[string]*GGUFMeta{}
	ggufMetaCacheMu sync.Mutex
)

// GGUFMetadataCached returns the parsed metadata for a GGUF file, backed by
// a process-local cache. Returns (nil, nil) when the file is not a GGUF.
func GGUFMetadataCached(path string) (*GGUFMeta, error) {
	ggufMetaCacheMu.Lock()
	if m, ok := ggufMetaCache[path]; ok {
		ggufMetaCacheMu.Unlock()
		return m, nil
	}
	ggufMetaCacheMu.Unlock()

	m, err := readGGUFMetadata(path)
	if m != nil {
		ggufMetaCacheMu.Lock()
		ggufMetaCache[path] = m
		ggufMetaCacheMu.Unlock()
	}
	return m, err
}

func readGGUFMetadata(path string) (*GGUFMeta, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()
	r := bufio.NewReader(f)

	var magic [4]byte
	if _, err := io.ReadFull(r, magic[:]); err != nil {
		return nil, fmt.Errorf("reading magic: %w", err)
	}
	if string(magic[:]) != ggufMagic {
		return nil, nil
	}

	var version uint32
	if err := binary.Read(r, binary.LittleEndian, &version); err != nil {
		return nil, fmt.Errorf("reading version: %w", err)
	}

	var tensorCount uint64
	if err := binary.Read(r, binary.LittleEndian, &tensorCount); err != nil {
		return nil, fmt.Errorf("reading tensor count: %w", err)
	}
	_ = tensorCount

	var metadataCount uint64
	if err := binary.Read(r, binary.LittleEndian, &metadataCount); err != nil {
		return nil, fmt.Errorf("reading metadata count: %w", err)
	}

	meta := &GGUFMeta{}

	for i := uint64(0); i < metadataCount; i++ {
		// Some GGUF writers add null-byte padding between metadata entries.
		// Skip leading null bytes before reading the key.
		for {
			b, err := r.ReadByte()
			if err != nil {
				return nil, fmt.Errorf("reading metadata key: %w", err)
			}
			if b != 0 {
				r.UnreadByte()
				break
			}
		}
		key, err := readGGUFString(r)
		if err != nil {
			return nil, fmt.Errorf("reading metadata key: %w", err)
		}

		var valueType uint32
		if err := binary.Read(r, binary.LittleEndian, &valueType); err != nil {
			return nil, fmt.Errorf("reading value type for %q: %w", key, err)
		}

		switch key {
		case "general.architecture":
			if valueType == 8 {
				val, err := readGGUFString(r)
				if err != nil {
					return nil, fmt.Errorf("reading architecture: %w", err)
				}
				meta.Architecture = val
				continue
			}
		case "general.file_type":
			if valueType == 5 {
				var val int32
				if err := binary.Read(r, binary.LittleEndian, &val); err != nil {
					return nil, fmt.Errorf("reading file_type: %w", err)
				}
				meta.FileType = uint32(val)
				if name, ok := ggufTypeNames[uint32(val)]; ok {
					meta.Quantization = name
				} else {
					meta.Quantization = fmt.Sprintf("type_%d", val)
				}
				continue
			}
		default:
			if meta.Architecture != "" && key == meta.Architecture+".block_count" {
				switch valueType {
				case 5: // INT32 (standard)
					var val int32
					if err := binary.Read(r, binary.LittleEndian, &val); err != nil {
						return nil, fmt.Errorf("reading block_count: %w", err)
					}
					meta.BlockCount = uint32(val)
					continue
				case 0: // UINT8 (defensive: some writers)
					var val uint8
					if err := binary.Read(r, binary.LittleEndian, &val); err != nil {
						return nil, fmt.Errorf("reading block_count: %w", err)
					}
					meta.BlockCount = uint32(val)
					continue
				}
			}
			if (key == "llama.context_length" || key == "gemma.context_length" ||
				key == "qwen2.context_length" || key == "starcoder2.context_length" ||
				key == "command-r.context_length" || key == "bert.context_length" ||
				key == "nomic-bert.context_length" || key == "mpt.context_length" ||
				key == "falcon.context_length" || key == "baichuan.context_length" ||
				key == "xverse.context_length" || key == "phi2.context_length" ||
				key == "phi3.context_length" || key == "stablelm.context_length" ||
				strings.HasSuffix(key, ".context_length")) && valueType == 10 {
				var val uint64
				if err := binary.Read(r, binary.LittleEndian, &val); err != nil {
					return nil, fmt.Errorf("reading context_length for %q: %w", key, err)
				}
				meta.ContextLength = val
				continue
			}
		}

		if err := skipGGUFValue(r, valueType); err != nil {
			return nil, fmt.Errorf("skipping value for %q: %w", key, err)
		}
	}

	return meta, nil
}

func readGGUFString(r io.Reader) (string, error) {
	var length uint64
	if err := binary.Read(r, binary.LittleEndian, &length); err != nil {
		return "", err
	}
	if length > 1<<20 {
		return "", fmt.Errorf("string too long: %d", length)
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return "", err
	}
	// Some GGUF writers include null terminators inside length-prefixed strings
	s := string(buf)
	s = strings.TrimLeft(s, "\x00")
	s = strings.TrimRight(s, "\x00")
	return s, nil
}

func skipGGUFValue(r io.Reader, valueType uint32) error {
	switch valueType {
	case 0, 1, 7: // UINT8, INT8, BOOL
		_, err := io.CopyN(io.Discard, r, 1)
		return err
	case 2, 3, 12, 13: // UINT16, INT16, FLOAT16, BF16
		_, err := io.CopyN(io.Discard, r, 2)
		return err
	case 4, 5: // FLOAT32, INT32
		_, err := io.CopyN(io.Discard, r, 4)
		return err
	case 6, 10, 11: // FLOAT64, UINT64, INT64
		_, err := io.CopyN(io.Discard, r, 8)
		return err
	case 8: // STRING
		_, err := readGGUFString(r)
		return err
	case 9: // ARRAY
		var elemType uint32
		if err := binary.Read(r, binary.LittleEndian, &elemType); err != nil {
			return err
		}
		var length uint64
		if err := binary.Read(r, binary.LittleEndian, &length); err != nil {
			return err
		}
		for i := uint64(0); i < length; i++ {
			if err := skipGGUFValue(r, elemType); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("unknown value type: %d", valueType)
	}
}

func deriveShortName(path string) string {
	base := filepath.Base(path)
	base = strings.TrimSuffix(base, ".gguf")

	// Remove split suffix e.g. "-00001-of-00005"
	re := regexp.MustCompile(`-\d{5}-of-\d{5}$`)
	base = re.ReplaceAllString(base, "")

	// Remove quantization suffix (single source of truth: StripQuantSuffix)
	return strings.ToLower(StripQuantSuffix(base))
}

// StripQuantSuffix removes a trailing quantization tag from a model name or
// GGUF filename stem (e.g. "Qwen2-0.5B-Instruct-Q4_K_M.gguf" →
// "Qwen2-0.5B-Instruct", "model-UD-Q4_K_M" → "model"). It is the single
// source of truth for quant-suffix stripping (P5-T2), replacing three
// divergent regexes. Matching is case-insensitive, understands vendor
// prefixes (UD-, ARM-), and works whether or not underscores have been
// normalized to hyphens elsewhere in the name. A no-match returns the input
// unchanged.
func StripQuantSuffix(name string) string {
	base := strings.TrimSuffix(name, ".gguf")
	lowerBase := strings.ToLower(base)
	for _, q := range knownQuantSuffixes {
		ql := strings.ToLower(q)
		if strings.HasSuffix(lowerBase, "-"+ql) || strings.HasSuffix(lowerBase, "_"+ql) {
			return base[:len(base)-len(ql)-1]
		}
	}
	return name
}

// DeriveShortNameFromRepo creates a clean API name from a HuggingFace model ID.
// e.g. "unsloth/gemma-4-12b-it-GGUF" → "gemma-4-12b-it"
func DeriveShortNameFromRepo(modelID string) string {
	// Take the last segment (e.g. "gemma-4-12b-it-GGUF" from "unsloth/gemma-4-12b-it-GGUF")
	parts := strings.Split(modelID, "/")
	base := parts[len(parts)-1]
	// Strip common repo suffixes
	base = regexp.MustCompile(`(?i)-gguf$|-instruct$|-hf$`).ReplaceAllString(base, "")
	return strings.ToLower(base)
}

func populateModelInfo(info *ModelInfo) error {
	meta, err := readGGUFMetadata(info.BlobPath)
	if err != nil {
		return err
	}

	changed := false
	if meta != nil {
		if meta.Architecture != "" && info.Architecture == "" {
			info.Architecture = meta.Architecture
			changed = true
		}
		if meta.Quantization != "" && info.Quantization == "" {
			info.Quantization = meta.Quantization
			changed = true
		}
		if meta.ContextLength > 0 && info.ContextLength == 0 {
			info.ContextLength = meta.ContextLength
			changed = true
		}
		if meta.BlockCount > 0 && info.BlockCount == 0 {
			info.BlockCount = meta.BlockCount
			changed = true
		}
	}

	// Derive short name from HF repo path when available (always refreshes for consistency)
	if strings.HasPrefix(info.Name, "hf.co/") {
		parts := strings.SplitN(strings.TrimPrefix(info.Name, "hf.co/"), ":", 2)
		derived := DeriveShortNameFromRepo(parts[0])
		if derived != info.ShortName {
			info.ShortName = derived
			changed = true
		}
	} else if info.ShortName == "" && info.BlobPath != "" {
		info.ShortName = deriveShortName(info.BlobPath)
		changed = true
	}

	if changed {
		if err := UpdateIndex(func(idx map[string]ModelInfo) error {
			if existing, ok := idx[info.Name]; ok {
				existing.Architecture = info.Architecture
				existing.Quantization = info.Quantization
				existing.ContextLength = info.ContextLength
				existing.ShortName = info.ShortName
				idx[info.Name] = existing
			}
			return nil
		}); err != nil {
			log.Printf("warning: could not save model metadata for %s: %v", info.Name, err)
		}
	}

	return nil
}
