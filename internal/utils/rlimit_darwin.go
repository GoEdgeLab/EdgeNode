//go:build darwin
// +build darwin

package utils

import (
	"syscall"
)

// SetRLimit set resource limit
func SetRLimit(limit uint64) error {
	var rLimit = &syscall.Rlimit{}
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, rLimit)
	if err != nil {
		return err
	}

	if rLimit.Cur < limit {
		rLimit.Cur = limit
	}
	if rLimit.Max < limit {
		rLimit.Max = limit
	}
	return syscall.Setrlimit(syscall.RLIMIT_NOFILE, rLimit)
}

// SetSuitableRLimit set best resource limit value
func SetSuitableRLimit() error {
	return SetRLimit(4096 * 100) // 1M=100Files
}
