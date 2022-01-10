// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package iplibrary

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils"
)

// AllowIP 检查IP是否被允许访问
// 如果一个IP不在任何名单中，则允许访问
func AllowIP(ip string, serverId int64) bool {
	var ipLong = utils.IP2Long(ip)
	if ipLong == 0 {
		return false
	}

	// check white lists
	if GlobalWhiteIPList.Contains(ipLong) {
		return true
	}

	if serverId > 0 {
		var list = SharedServerListManager.FindWhiteList(serverId, false)
		if list != nil && list.Contains(ipLong) {
			return true
		}
	}

	// check black lists
	if GlobalBlackIPList.Contains(ipLong) {
		return false
	}

	if serverId > 0 {
		var list = SharedServerListManager.FindBlackList(serverId, false)
		if list != nil && list.Contains(ipLong) {
			return false
		}
	}

	return true
}

// IsInWhiteList 检查IP是否在白名单中
func IsInWhiteList(ip string) bool {
	var ipLong = utils.IP2Long(ip)
	if ipLong == 0 {
		return false
	}

	// check white lists
	return GlobalWhiteIPList.Contains(ipLong)
}

// AllowIPStrings 检查一组IP是否被允许访问
func AllowIPStrings(ipStrings []string, serverId int64) bool {
	if len(ipStrings) == 0 {
		return true
	}
	for _, ip := range ipStrings {
		isAllowed := AllowIP(ip, serverId)
		if !isAllowed {
			return false
		}
	}
	return true
}
