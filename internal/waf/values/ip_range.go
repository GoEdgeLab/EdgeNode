// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package values

import (
	"bytes"
	"net"
	"strings"
)

type IPRangeType = string

const (
	IPRangeTypeCIDR    IPRangeType = "cidr"     // CIDR
	IPRangeTypeSingeIP IPRangeType = "singleIP" // 单个IP
	IPRangeTypeRange   IPRangeType = "range"    // IP范围，IP1-IP2
)

type IPRange struct {
	Type   IPRangeType
	CIDR   *net.IPNet
	IPFrom net.IP
	IPTo   net.IP
}

func (this *IPRange) Contains(netIP net.IP) bool {
	if netIP == nil {
		return false
	}
	switch this.Type {
	case IPRangeTypeCIDR:
		if this.CIDR != nil {
			return this.CIDR.Contains(netIP)
		}
	case IPRangeTypeSingeIP:
		if this.IPFrom != nil {
			return this.IPFrom.Equal(netIP)
		}
	case IPRangeTypeRange:
		return bytes.Compare(this.IPFrom, netIP) <= 0 && bytes.Compare(this.IPTo, netIP) >= 0
	}
	return false
}

type IPRangeList struct {
	Ranges []*IPRange
}

func NewIPRangeList() *IPRangeList {
	return &IPRangeList{}
}

func ParseIPRangeList(value string) *IPRangeList {
	var list = NewIPRangeList()

	if len(value) == 0 {
		return list
	}

	var lines = strings.Split(value, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		if strings.Contains(line, ",") { // IPFrom,IPTo
			var pieces = strings.SplitN(line, ",", 2)
			if len(pieces) == 2 {
				var ipFrom = net.ParseIP(strings.TrimSpace(pieces[0]))
				var ipTo = net.ParseIP(strings.TrimSpace(pieces[1]))
				if ipFrom != nil && ipTo != nil {
					if bytes.Compare(ipFrom, ipTo) > 0 {
						ipFrom, ipTo = ipTo, ipFrom
					}
					list.Ranges = append(list.Ranges, &IPRange{
						Type:   IPRangeTypeRange,
						IPFrom: ipFrom,
						IPTo:   ipTo,
					})
				}
			}
		} else if strings.Contains(line, "-") { // IPFrom-IPTo
			var pieces = strings.SplitN(line, "-", 2)
			if len(pieces) == 2 {
				var ipFrom = net.ParseIP(strings.TrimSpace(pieces[0]))
				var ipTo = net.ParseIP(strings.TrimSpace(pieces[1]))
				if ipFrom != nil && ipTo != nil {
					if bytes.Compare(ipFrom, ipTo) > 0 {
						ipFrom, ipTo = ipTo, ipFrom
					}
					list.Ranges = append(list.Ranges, &IPRange{
						Type:   IPRangeTypeRange,
						IPFrom: ipFrom,
						IPTo:   ipTo,
					})
				}
			}
		} else if strings.Contains(line, "/") { // CIDR
			_, cidr, _ := net.ParseCIDR(line)
			if cidr != nil {
				list.Ranges = append(list.Ranges, &IPRange{
					Type: IPRangeTypeCIDR,
					CIDR: cidr,
				})
			}
		} else { // single ip
			var netIP = net.ParseIP(line)
			if netIP != nil {
				list.Ranges = append(list.Ranges, &IPRange{
					Type:   IPRangeTypeSingeIP,
					IPFrom: netIP,
				})
			}
		}
	}

	return list
}

func (this *IPRangeList) Contains(ip string) bool {
	var netIP = net.ParseIP(ip)
	if netIP == nil {
		return false
	}
	for _, r := range this.Ranges {
		if r.Contains(netIP) {
			return true
		}
	}
	return false
}
