// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package firewalls

import (
	"github.com/iwind/TeaGo/types"
	"strings"
	"sync"
	"time"
)

type BaseFirewall struct {
	locker        sync.Mutex
	latestIPTimes []string // [ip@time, ....]
}

// 检查是否在最近添加过
func (this *BaseFirewall) checkLatestIP(ip string) bool {
	this.locker.Lock()
	defer this.locker.Unlock()

	var expiredIndex = -1
	for index, ipTime := range this.latestIPTimes {
		var pieces = strings.Split(ipTime, "@")
		var oldIP = pieces[0]
		var oldTimestamp = pieces[1]
		if types.Int64(oldTimestamp) < time.Now().Unix()-3 /** 3秒外表示过期 **/ {
			expiredIndex = index
			continue
		}
		if oldIP == ip {
			return true
		}
	}

	if expiredIndex > -1 {
		this.latestIPTimes = this.latestIPTimes[expiredIndex+1:]
	}

	this.latestIPTimes = append(this.latestIPTimes, ip+"@"+types.String(time.Now().Unix()))
	const maxLen = 128
	if len(this.latestIPTimes) > maxLen {
		this.latestIPTimes = this.latestIPTimes[1:]
	}

	return false
}
