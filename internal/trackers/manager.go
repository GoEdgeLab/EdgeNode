// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package trackers

import (
	"sync"
)

var SharedManager = NewManager()

type Manager struct {
	m      map[string][]float64 // label => time costs ms
	locker sync.Mutex
}

func NewManager() *Manager {
	return &Manager{m: map[string][]float64{}}
}

func (this *Manager) Add(label string, costMs float64) {
	this.locker.Lock()
	costs, ok := this.m[label]
	if ok {
		costs = append(costs, costMs)
		if len(costs) > 5 { // 只取最近的N条
			costs = costs[1:]
		}
		this.m[label] = costs
	} else {
		this.m[label] = []float64{costMs}
	}
	this.locker.Unlock()
}

func (this *Manager) Labels() map[string]float64 {
	var result = map[string]float64{}
	this.locker.Lock()
	for label, costs := range this.m {
		var sum float64
		for _, cost := range costs {
			sum += cost
		}
		result[label] = sum / float64(len(costs))
	}
	this.locker.Unlock()
	return result
}
