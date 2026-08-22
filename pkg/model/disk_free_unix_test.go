//go:build !windows

package model

import (
	"path/filepath"
	"testing"
)

// TestFreeDiskBytesPopulated (P4-T1 fix): freeDiskBytes must report a
// plausible (non-zero) free-space figure on a populated filesystem. On
// filesystems that report st_bsize=0 (some tmpfs), the old Bsize-only math
// returned 0 bytes free and blocked every pull.
func TestFreeDiskBytesPopulated(t *testing.T) {
	dir := t.TempDir()

	got, err := freeDiskBytes(dir)
	if err != nil {
		t.Fatalf("freeDiskBytes: %v", err)
	}
	if got == 0 {
		t.Fatal("freeDiskBytes = 0 on a populated filesystem — block-size math is wrong")
	}
}

// TestFreeDiskBytesMissingPath verifies the error path.
func TestFreeDiskBytesMissingPath(t *testing.T) {
	if _, err := freeDiskBytes(filepath.Join(t.TempDir(), "does-not-exist")); err == nil {
		t.Fatal("expected an error for a missing path")
	}
}
