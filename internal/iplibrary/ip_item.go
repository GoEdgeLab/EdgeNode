package iplibrary

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils/fasttime"
)

type IPItemType = string

const (
	IPItemTypeIPv4 IPItemType = "ipv4" // IPv4
	IPItemTypeIPv6 IPItemType = "ipv6" // IPv6
	IPItemTypeAll  IPItemType = "all"  // 所有IP
)

// IPItem IP条目
type IPItem struct {
	Type   string `json:"type"`
	Id     uint64 `json:"id"`
	IPFrom []byte `json:"ipFrom"`
	IPTo   []byte `json:"ipTo"`

	ExpiredAt  int64  `json:"expiredAt"`
	EventLevel string `json:"eventLevel"`
}

// Contains 检查是否包含某个IP
func (this *IPItem) Contains(ipBytes []byte) bool {
	switch this.Type {
	case IPItemTypeIPv4:
		return this.containsIP(ipBytes)
	case IPItemTypeIPv6:
		return this.containsIP(ipBytes)
	case IPItemTypeAll:
		return this.containsAll()
	default:
		return this.containsIP(ipBytes)
	}
}

// 检查是否包含某个
func (this *IPItem) containsIP(ipBytes []byte) bool {
	if IsZero(this.IPTo) {
		if CompareBytes(this.IPFrom, ipBytes) != 0 {
			return false
		}
	} else {
		if CompareBytes(this.IPFrom, ipBytes) > 0 || CompareBytes(this.IPTo, ipBytes) < 0 {
			return false
		}
	}
	if this.ExpiredAt > 0 && this.ExpiredAt < fasttime.Now().Unix() {
		return false
	}
	return true
}

// 检查是否包所有IP
func (this *IPItem) containsAll() bool {
	if this.ExpiredAt > 0 && this.ExpiredAt < fasttime.Now().Unix() {
		return false
	}
	return true
}
