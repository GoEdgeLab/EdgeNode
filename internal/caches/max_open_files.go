// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package caches

import (
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"sync/atomic"
	"time"
)

const (
	modeSlow int32 = 1
	modeFast int32 = 2
)

// MaxOpenFiles max open files manager
type MaxOpenFiles struct {
	ticker *time.Ticker
	mode   int32
}

func NewMaxOpenFiles() *MaxOpenFiles {
	var f = &MaxOpenFiles{}
	f.ticker = time.NewTicker(1 * time.Second)
	f.init()
	return f
}

func (this *MaxOpenFiles) init() {
	goman.New(func() {
		for range this.ticker.C {
			// reset mode
			atomic.StoreInt32(&this.mode, modeFast)
		}
	})
}

func (this *MaxOpenFiles) Fast() {
	atomic.AddInt32(&this.mode, modeFast)
}

func (this *MaxOpenFiles) FinishAll() {
	this.Fast()
}

func (this *MaxOpenFiles) Slow() {
	atomic.StoreInt32(&this.mode, modeSlow)
}

func (this *MaxOpenFiles) Next() bool {
	return atomic.LoadInt32(&this.mode) != modeSlow
}
