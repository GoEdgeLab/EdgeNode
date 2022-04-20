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

	modSlow int32 = 1
	modFast int32 = 2
)

type MaxOpenFiles struct {
	step         int32
	maxOpenFiles int32
	ptr          *int32
	ticker       *time.Ticker
	mod          int32
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
	goman.New(func() {
		for range f.ticker.C {
			var mod = atomic.LoadInt32(&f.mod)
			switch mod {
			case modSlow:
				// we decrease more quickly, with more steps
				if atomic.AddInt32(f.ptr, -step*2) <= 0 {
					atomic.StoreInt32(f.ptr, minOpenFilesValue)
				}
			case modFast:
				if atomic.AddInt32(f.ptr, step) >= maxOpenFilesValue {
					atomic.StoreInt32(f.ptr, maxOpenFilesValue)
				}
			}

			// reset mod
			atomic.StoreInt32(&f.mod, 0)
		}
	})
	return f
}

func (this *MaxOpenFiles) Fast() {
	if atomic.LoadInt32(&this.mod) == 0 {
		this.mod = modFast
	}
}

func (this *MaxOpenFiles) Slow() {
	atomic.StoreInt32(&this.mod, modSlow)
}

func (this *MaxOpenFiles) Max() int32 {
	if atomic.LoadInt32(&this.mod) == modSlow {
		return 0
	}

	var v = atomic.LoadInt32(this.ptr)
	if v <= minOpenFilesValue {
		return minOpenFilesValue
	}
	return v
}
