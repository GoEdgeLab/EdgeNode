// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package ratelimit

import (
	"github.com/TeaOSLab/EdgeNode/internal/zero"
	"sync"
)

type Counter struct {
	count     int
	sem       chan zero.Zero
	done      chan zero.Zero
	closeOnce sync.Once
}

func NewCounter(count int) *Counter {
	return &Counter{
		count: count,
		sem:   make(chan zero.Zero, count),
		done:  make(chan zero.Zero),
	}
}

func (this *Counter) Count() int {
	return this.count
}

// Len 已占用数量
func (this *Counter) Len() int {
	return len(this.sem)
}

func (this *Counter) Ack() bool {
	select {
	case this.sem <- zero.New():
		return true
	case <-this.done:
		return false
	}
}

func (this *Counter) Release() {
	select {
	case <-this.sem:
	default:
		// 总是能Release成功
	}
}

func (this *Counter) Close() {
	this.closeOnce.Do(func() {
		close(this.done)
	})
}
