// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package metrics

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"sync"
	"sync/atomic"
)

type BaseTask struct {
	itemConfig *serverconfigs.MetricItemConfig
	isLoaded   bool
	isStopped  bool

	statsMap    map[string]*Stat // 待写入队列，hash => *Stat
	statsLocker sync.RWMutex
}

// Add 添加数据
func (this *BaseTask) Add(obj MetricInterface) {
	if this.isStopped || !this.isLoaded {
		return
	}

	var keys = []string{}
	for _, key := range this.itemConfig.Keys {
		var k = obj.MetricKey(key)

		// 忽略499状态
		if key == "${status}" && k == "499" {
			return
		}

		keys = append(keys, k)
	}

	v, ok := obj.MetricValue(this.itemConfig.Value)
	if !ok {
		return
	}

	var hash = UniqueKey(obj.MetricServerId(), keys, this.itemConfig.CurrentTime(), this.itemConfig.Version, this.itemConfig.Id)
	var countItems int
	this.statsLocker.RLock()
	oldStat, ok := this.statsMap[hash]
	if !ok {
		countItems = len(this.statsMap)
	}
	this.statsLocker.RUnlock()
	if ok {
		atomic.AddInt64(&oldStat.Value, 1)
	} else {
		// 防止过载
		if countItems < MaxQueueSize {
			this.statsLocker.Lock()
			this.statsMap[hash] = &Stat{
				ServerId: obj.MetricServerId(),
				Keys:     keys,
				Value:    v,
				Time:     this.itemConfig.CurrentTime(),
				Hash:     hash,
			}
			this.statsLocker.Unlock()
		}
	}
}

func (this *BaseTask) Item() *serverconfigs.MetricItemConfig {
	return this.itemConfig
}

func (this *BaseTask) SetItem(itemConfig *serverconfigs.MetricItemConfig) {
	this.itemConfig = itemConfig
}
