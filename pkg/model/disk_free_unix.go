//go:build !windows && !linux

package model

import "syscall"

// On macOS/BSD, st_f_bsize is already the fundamental block size, so Bsize is
// the correct multiplier for st_bavail.
func freeDiskBytes(path string) (uint64, error) {
	var fs syscall.Statfs_t
	if err := syscall.Statfs(path, &fs); err != nil {
		return 0, err
	}
	bavail := uint64(int64(fs.Bavail))
	return bavail * uint64(fs.Bsize), nil
}
