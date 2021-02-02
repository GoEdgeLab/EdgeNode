package utils

import (
	"strings"
)

// 计算版本代号
func VersionToLong(version string) uint32 {
	countDots := strings.Count(version, ".")
	if countDots == 2 {
		version += ".0"
	} else if countDots == 1 {
		version += ".0.0"
	} else if countDots == 0 {
		version += ".0.0.0"
	}
	return uint32(IP2Long(version))
}
