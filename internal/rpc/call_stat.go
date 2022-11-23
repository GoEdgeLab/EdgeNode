// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package rpc

import (
	"sync"
)

type callStatItem struct {
	ok          bool
	costSeconds float64
}

type CallStat struct {
	size  int
	items []*callStatItem

	locker sync.Mutex
}

func NewCallStat(size int) *CallStat {
	return &CallStat{
		size: size,
	}
}

func (this *CallStat) Add(ok bool, costSeconds float64) {
	var size = this.size
	if size <= 0 {
		size = 10
	}

	this.locker.Lock()
	this.items = append(this.items, &callStatItem{
		ok:          ok,
		costSeconds: costSeconds,
	})
	if len(this.items) > size {
		this.items = this.items[1:]
	}
	this.locker.Unlock()
}

func (this *CallStat) Sum() (successPercent float64, avgCostSeconds float64) {
	this.locker.Lock()
	defer this.locker.Unlock()

	var size = this.size
	if size <= 0 {
		size = 10
	}

	var totalItems = len(this.items)
	if totalItems <= size/2 /** 低于一半的采样率，不计入统计 **/ {
		successPercent = 100
		return
	}

	var totalOkItems = 0
	var totalCostSeconds float64
	for _, item := range this.items {
		if item.ok {
			totalOkItems++
		}
		totalCostSeconds += item.costSeconds
	}
	successPercent = float64(totalOkItems) * 100 / float64(totalItems)
	avgCostSeconds = totalCostSeconds / float64(totalItems)

	return
}
