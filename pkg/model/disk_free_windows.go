//go:build windows

package model

func freeDiskBytes(path string) (uint64, error) {
	return 0, nil
}
