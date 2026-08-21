//go:build !linux

package manager

// cpuTicksPerSec/procCPUTicks are the Linux /proc/<pid>/stat CPU samplers
// (P2-T7); on other OSes the metrics goroutine uses ps.
func cpuTicksPerSec() float64 { return 0 }

func procCPUTicks(pid int) (float64, bool) { return 0, false }
