// +build darwin

package utils

import (
	"syscall"
)

const SO_REUSEPORT = syscall.SO_REUSEPORT
