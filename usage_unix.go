//go:build unix

package main

import (
	"os"
	"runtime"
	"syscall"
)

// processUsage extracts CPU seconds and peak RSS in GB from a finished child.
// wait4 reports the resources of that child and of every descendant it reaped,
// so nothing inside the process tree has to cooperate.
func processUsage(state *os.ProcessState) (cpuS, rssGB float64) {
	usage, ok := state.SysUsage().(*syscall.Rusage)
	if !ok {
		return state.UserTime().Seconds() + state.SystemTime().Seconds(), 0
	}
	cpuS = float64(usage.Utime.Nano()+usage.Stime.Nano()) / 1e9
	// getrusage reports kilobytes on Linux and bytes on macOS.
	divisor := 1024.0 * 1024.0
	if runtime.GOOS == "darwin" {
		divisor = 1024 * 1024 * 1024
	}
	return cpuS, float64(usage.Maxrss) / divisor
}
