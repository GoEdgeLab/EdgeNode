// Copyright 2023 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package nftables

import (
	"sync"
	"time"
)

type Expiration struct {
	m map[string]time.Time // key => expires time

	lastGCAt int64

	locker sync.RWMutex
}

func NewExpiration() *Expiration {
	return &Expiration{
		m: map[string]time.Time{},
	}
}

func (this *Expiration) AddUnsafe(key []byte, expires time.Time) {
	this.m[string(key)] = expires
}

func (this *Expiration) Add(key []byte, expires time.Time) {
	this.locker.Lock()
	this.m[string(key)] = expires
	this.gc()
	this.locker.Unlock()
}

func (this *Expiration) Remove(key []byte) {
	this.locker.Lock()
	delete(this.m, string(key))
	this.locker.Unlock()
}

func (this *Expiration) Contains(key []byte) bool {
	this.locker.RLock()
	expires, ok := this.m[string(key)]
	if ok && expires.Year() > 2000 && time.Now().After(expires) {
		ok = false
	}
	this.locker.RUnlock()
	return ok
}

func (this *Expiration) gc() {
	// we won't gc too frequently
	var currentTime = time.Now().Unix()
	if this.lastGCAt >= currentTime {
		return
	}
	this.lastGCAt = currentTime

	var now = time.Now().Add(-10 * time.Second) // gc elements expired before 10 seconds ago
	for key, expires := range this.m {
		if expires.Year() > 2000 && now.After(expires) {
			delete(this.m, key)
		}
	}
}
