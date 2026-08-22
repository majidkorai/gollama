//go:build linux

package model

import "syscall"

// On Linux, st_bavail is counted in units of st_frsize (not st_bsize). Some
// filesystems (e.g. tmpfs on certain kernels) report st_bsize=0, which would
// make the old Bsize-based math return 0 bytes free and block every pull.
// Prefer Frsize, falling back to Bsize for filesystems that populate it instead.
func freeDiskBytes(path string) (uint64, error) {
	var fs syscall.Statfs_t
	if err := syscall.Statfs(path, &fs); err != nil {
		return 0, err
	}
	bavail := uint64(int64(fs.Bavail))
	bsize := uint64(fs.Frsize)
	if bsize == 0 {
		bsize = uint64(fs.Bsize)
	}
	return bavail * bsize, nil
}
