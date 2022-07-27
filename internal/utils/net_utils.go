// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package utils

import (
	"net"
	"sort"
)

// ParseAddrHost 分析地址中的主机名部分
func ParseAddrHost(addr string) string {
	if len(addr) == 0 {
		return addr
	}

	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return host
}

// MergePorts 聚合端口
// 返回 [ [fromPort, toPort], ... ]
func MergePorts(ports []int) [][2]int {
	if len(ports) == 0 {
		return nil
	}

	sort.Ints(ports)

	var result = [][2]int{}
	var lastRange = [2]int{0, 0}
	var lastPort = -1
	for _, port := range ports {
		if port <= 0 /** 只处理有效的端口 **/ || port == lastPort /** 去重 **/ {
			continue
		}

		if lastPort < 0 || port != lastPort+1 {
			lastRange = [2]int{port, port}
			result = append(result, lastRange)
		} else { // 如果是连续的
			lastRange[1] = port
			result[len(result)-1] = lastRange
		}

		lastPort = port
	}

	return result
}
