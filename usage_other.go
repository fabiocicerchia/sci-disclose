//go:build !unix

package main

import "os"

// processUsage falls back to the coarser CPU times the runtime exposes
// everywhere; peak RSS is not available, so memory must be declared.
func processUsage(state *os.ProcessState) (cpuS, rssGB float64) {
	return state.UserTime().Seconds() + state.SystemTime().Seconds(), 0
}
