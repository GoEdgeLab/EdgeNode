// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package expires

import (
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"sync"
	"time"
)

var SharedManager = NewManager()

type Manager struct {
	listMap map[*List]bool
	locker  sync.Mutex
	ticker  *time.Ticker
}

func NewManager() *Manager {
	var manager = &Manager{
		listMap: map[*List]bool{},
		ticker:  time.NewTicker(1 * time.Second),
	}
	goman.New(func() {
		manager.init()
	})
	return manager
}

func (this *Manager) init() {
	var lastTimestamp = int64(0)
	for range this.ticker.C {
		timestamp := time.Now().Unix()
		if lastTimestamp == 0 {
			lastTimestamp = timestamp - 3600
		}

		if timestamp >= lastTimestamp {
			for i := lastTimestamp; i <= timestamp; i++ {
				this.locker.Lock()
				for list := range this.listMap {
					list.GC(i, list.gcCallback)
				}
				this.locker.Unlock()
			}
		} else {
			for i := timestamp; i <= lastTimestamp; i++ {
				this.locker.Lock()
				for list := range this.listMap {
					list.GC(i, list.gcCallback)
				}
				this.locker.Unlock()
			}
		}

		// 这样做是为了防止系统时钟突变
		lastTimestamp = timestamp
	}
}

func (this *Manager) Add(list *List) {
	this.locker.Lock()
	this.listMap[list] = true
	this.locker.Unlock()
}

func (this *Manager) Remove(list *List) {
	this.locker.Lock()
	delete(this.listMap, list)
	this.locker.Unlock()
}
