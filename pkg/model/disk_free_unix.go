//go:build !windows

package model

import "syscall"

func freeDiskBytes(path string) (uint64, error) {
	var fs syscall.Statfs_t
	if err := syscall.Statfs(path, &fs); err != nil {
		return 0, err
	}
	bavail := uint64(int64(fs.Bavail))
	bsize := uint64(fs.Bsize)
	return bavail * bsize, nil
}
