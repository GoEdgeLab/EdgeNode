// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package caches

import (
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"sync/atomic"
	"time"
)

const (
	minOpenFilesValue int32 = 2
	maxOpenFilesValue int32 = 65535

	modeSlow int32 = 1
	modeFast int32 = 2
)

// MaxOpenFiles max open files manager
type MaxOpenFiles struct {
	step         int32
	maxOpenFiles int32
	ptr          *int32
	ticker       *time.Ticker
	mode         int32

	lastOpens    int32
	currentOpens int32
}

func NewMaxOpenFiles(step int32) *MaxOpenFiles {
	if step <= 0 {
		step = 2
	}
	var f = &MaxOpenFiles{
		step:         step,
		maxOpenFiles: 2,
	}
	if teaconst.DiskIsFast {
		f.maxOpenFiles = 32
	}
	f.ptr = &f.maxOpenFiles
	f.ticker = time.NewTicker(1 * time.Second)
	f.init()
	return f
}

func (this *MaxOpenFiles) init() {
	goman.New(func() {
		for range this.ticker.C {
			var mod = atomic.LoadInt32(&this.mode)
			switch mod {
			case modeSlow:
				// we decrease more quickly, with more steps
				if atomic.AddInt32(this.ptr, -this.step*2) <= 0 {
					atomic.StoreInt32(this.ptr, minOpenFilesValue)
				}
			case modeFast:
				// we increase only when file opens increases
				var currentOpens = atomic.LoadInt32(&this.currentOpens)
				if currentOpens > this.lastOpens {
					if atomic.AddInt32(this.ptr, this.step) >= maxOpenFilesValue {
						atomic.StoreInt32(this.ptr, maxOpenFilesValue)
					}
				}
				this.lastOpens = currentOpens
				atomic.StoreInt32(&this.currentOpens, 0)
			}

			// reset mode
			atomic.StoreInt32(&this.mode, 0)
		}
	})
}

func (this *MaxOpenFiles) Fast() {
	if atomic.LoadInt32(&this.mode) == 0 {
		this.mode = modeFast
	}
	atomic.AddInt32(&this.currentOpens, 1)
}

func (this *MaxOpenFiles) Slow() {
	atomic.StoreInt32(&this.mode, modeSlow)
}

func (this *MaxOpenFiles) Max() int32 {
	if atomic.LoadInt32(&this.mode) == modeSlow {
		return 0
	}

	var v = atomic.LoadInt32(this.ptr)
	if v <= minOpenFilesValue {
		return minOpenFilesValue
	}
	return v
}
