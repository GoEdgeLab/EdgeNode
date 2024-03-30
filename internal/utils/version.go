package utils

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/configutils"
	"strings"
)

// VersionToLong 计算版本代号
func VersionToLong(version string) uint32 {
	var countDots = strings.Count(version, ".")
	if countDots == 2 {
		version += ".0"
	} else if countDots == 1 {
		version += ".0.0"
	} else if countDots == 0 {
		version += ".0.0.0"
	}
	return uint32(configutils.IPString2Long(version))
}
