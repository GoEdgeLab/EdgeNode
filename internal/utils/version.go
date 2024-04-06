package utils

import (
	"encoding/binary"
	"net"
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
	var ip = net.ParseIP(version)
	if ip == nil || ip.To4() == nil {
		return 0
	}
	return binary.BigEndian.Uint32(ip.To4())
}
