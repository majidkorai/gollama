package model

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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

type ggufMetadata struct {
	Architecture   string
	Quantization   string
	ContextLength  uint64
	FileType       uint32
	BlockCount     uint32
}

func readGGUFMetadata(path string) (*ggufMetadata, error) {
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

	meta := &ggufMetadata{}

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
			if meta.Architecture != "" && key == meta.Architecture+".block_count" && valueType == 5 {
				var val int32
				if err := binary.Read(r, binary.LittleEndian, &val); err != nil {
					return nil, fmt.Errorf("reading block_count: %w", err)
				}
				meta.BlockCount = uint32(val)
				continue
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

	// Remove quantization suffix (common patterns)
	quantRe := regexp.MustCompile(`(?i)(-UD)?-[IQBF][QKBF][0-9]_[SLMX](_[SLMX])?$|-[BQKF][0-9]_[A-Z_]+$`)
	base = quantRe.ReplaceAllString(base, "")

	return strings.ToLower(base)
}

// ReadBlockCount reads the block count from a GGUF model file.
func ReadBlockCount(path string) uint32 {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()

	// Only read the first 10 MB — metadata is always near the start
	const maxHead = 10 * 1024 * 1024
	fi, err := f.Stat()
	if err != nil {
		return 0
	}
	headSize := fi.Size()
	if headSize > maxHead {
		headSize = maxHead
	}
	data := make([]byte, headSize)
	if _, err := io.ReadFull(f, data); err != nil && err != io.ErrUnexpectedEOF {
		return 0
	}

	// Scan for "general.architecture" in file bytes to find the arch name
	pat := []byte("general.architecture")
	idx := bytes.Index(data, pat)
	if idx < 0 {
		return 0
	}

	pos := idx + len(pat)
	if pos+12 > len(data) {
		return 0
	}
	valType := binary.LittleEndian.Uint32(data[pos : pos+4])
	pos += 4
	if valType != 8 {
		return 0
	}
	strLen := binary.LittleEndian.Uint64(data[pos : pos+8])
	pos += 8
	if strLen == 0 || pos+int(strLen) > len(data) {
		return 0
	}
	arch := string(data[pos : pos+int(strLen)])

	blockKey := []byte(arch + ".block_count")
	bkIdx := bytes.Index(data, blockKey)
	if bkIdx < 0 {
		return 0
	}

	bkValType := binary.LittleEndian.Uint32(data[bkIdx+len(blockKey) : bkIdx+len(blockKey)+4])
	switch bkValType {
	case 5, 4: // INT32 or FLOAT32 (some writers store ints as float type)
		return binary.LittleEndian.Uint32(data[bkIdx+len(blockKey)+4 : bkIdx+len(blockKey)+8])
	case 0:
		return uint32(data[bkIdx+len(blockKey)+4])
	}
	return 0
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
		UpdateIndex(func(idx map[string]ModelInfo) error {
			if existing, ok := idx[info.Name]; ok {
				existing.Architecture = info.Architecture
				existing.Quantization = info.Quantization
				existing.ContextLength = info.ContextLength
				existing.ShortName = info.ShortName
				idx[info.Name] = existing
			}
			return nil
		})
	}

	return nil
}
