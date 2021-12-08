// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package ttlcache

import (
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"sync"
	"time"
)

var SharedManager = NewManager()

type Manager struct {
	ticker *time.Ticker
	locker sync.Mutex

	cacheMap map[*Cache]bool
}

func NewManager() *Manager {
	var manager = &Manager{
		ticker:   time.NewTicker(3 * time.Second),
		cacheMap: map[*Cache]bool{},
	}

	goman.New(func() {
		manager.init()
	})

	return manager
}

func (this *Manager) init() {
	for range this.ticker.C {
		this.locker.Lock()
		for cache := range this.cacheMap {
			cache.GC()
		}
		this.locker.Unlock()
	}
}

func (this *Manager) Add(cache *Cache) {
	this.locker.Lock()
	this.cacheMap[cache] = true
	this.locker.Unlock()
}

func (this *Manager) Remove(cache *Cache) {
	this.locker.Lock()
	delete(this.cacheMap, cache)
	this.locker.Unlock()
}
