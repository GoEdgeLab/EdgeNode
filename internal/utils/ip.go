package utils

import (
	"encoding/binary"
	"github.com/cespare/xxhash"
	"math"
	"net"
	"strings"
)

// 将IP转换为整型
// 注意IPv6没有顺序
func IP2Long(ip string) uint64 {
	s := net.ParseIP(ip)
	if s == nil {
		return 0
	}

	if strings.Contains(ip, ":") {
		return math.MaxUint32 + xxhash.Sum64String(ip)
	}
	return uint64(binary.BigEndian.Uint32(s.To4()))
}
