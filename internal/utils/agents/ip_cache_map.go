// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package agents

import (
	"github.com/TeaOSLab/EdgeNode/internal/zero"
	"sync"
)

type IPCacheMap struct {
	m    map[string]zero.Zero
	list []string

	locker sync.RWMutex
	maxLen int
}

func NewIPCacheMap(maxLen int) *IPCacheMap {
	if maxLen <= 0 {
		maxLen = 65535
	}
	return &IPCacheMap{
		m:      map[string]zero.Zero{},
		maxLen: maxLen,
	}
}

func (this *IPCacheMap) Add(ip string) {
	this.locker.Lock()
	defer this.locker.Unlock()

	// 是否已经存在
	_, ok := this.m[ip]
	if ok {
		return
	}

	// 超出长度删除第一个
	if len(this.list) >= this.maxLen {
		delete(this.m, this.list[0])
		this.list = this.list[1:]
	}

	// 加入新数据
	this.m[ip] = zero.Zero{}
	this.list = append(this.list, ip)
}

func (this *IPCacheMap) Contains(ip string) bool {
	this.locker.RLock()
	defer this.locker.RUnlock()
	_, ok := this.m[ip]
	return ok
}
