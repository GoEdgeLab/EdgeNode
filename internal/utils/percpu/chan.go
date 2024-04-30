// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package percpu

import (
	"runtime"
)

type Chan[T any] struct {
	c chan T

	count int
	cList []chan T
}

func NewChan[T any](size int) *Chan[T] {
	var count = max(runtime.NumCPU(), runtime.GOMAXPROCS(0))
	var cList []chan T
	for i := 0; i < count; i++ {
		cList = append(cList, make(chan T, size))
	}

	return &Chan[T]{
		c:     make(chan T, size),
		count: count,
		cList: cList,
	}
}

func (this *Chan[T]) C() chan T {
	var procId = GetProcId()
	if procId < this.count {
		return this.cList[procId]
	}
	return this.c
}
