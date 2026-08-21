//go:build linux

package manager

import (
	"fmt"
	"os"
)

// cpuTicksPerSec returns the CPU clock frequency that /proc/<pid>/stat
// utime/stime are expressed in (P2-T7). The stdlib has no Sysconf, so this
// is the kernel's USER_HZ, which is 100 on every Linux system.
func cpuTicksPerSec() float64 {
	return 100
}

// procCPUTicks returns a process's utime+stime (in clock ticks) read from
// /proc/<pid>/stat (P2-T7).
func procCPUTicks(pid int) (float64, bool) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return 0, false
	}
	return parseProcStatTicks(data)
}
