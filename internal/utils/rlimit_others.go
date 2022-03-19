//go:build !linux && !darwin
// +build !linux,!darwin

package utils

// set resource limit
func SetRLimit(limit uint64) error {
	return nil
}

// set best resource limit value
func SetSuitableRLimit() error {
	return nil
}
