// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package expires

import (
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/TeaOSLab/EdgeNode/internal/zero"
	"sync"
	"time"
)

var SharedManager = NewManager()

type Manager struct {
	listMap map[*List]zero.Zero
	locker  sync.Mutex
	ticker  *time.Ticker
}

func NewManager() *Manager {
	var manager = &Manager{
		listMap: map[*List]zero.Zero{},
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
		var currentTime = time.Now().Unix()
		if lastTimestamp == 0 {
			lastTimestamp = currentTime - 86400 // prevent timezone changes
		}

		if currentTime >= lastTimestamp {
			for i := lastTimestamp; i <= currentTime; i++ {
				this.locker.Lock()
				for list := range this.listMap {
					list.GC(i)
				}
				this.locker.Unlock()
			}
		} else {
			// 如果过去的时间比现在大，则从这一秒重新开始
			for i := currentTime; i <= currentTime; i++ {
				this.locker.Lock()
				for list := range this.listMap {
					list.GC(i)
				}
				this.locker.Unlock()
			}
		}

		// 这样做是为了防止系统时钟突变
		lastTimestamp = currentTime
	}
}

func (this *Manager) Add(list *List) {
	this.locker.Lock()
	this.listMap[list] = zero.New()
	this.locker.Unlock()
}

func (this *Manager) Remove(list *List) {
	this.locker.Lock()
	delete(this.listMap, list)
	this.locker.Unlock()
}
