// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package zero

import (
	"runtime"
	"testing"
)

func TestZero_Chan(t *testing.T) {
	var stat1 = &runtime.MemStats{}
	runtime.ReadMemStats(stat1)

	var m = make(chan Zero, 2_000_000)
	for i := 0; i < 1_000_000; i++ {
		m <- New()
	}

	var stat2 = &runtime.MemStats{}
	runtime.ReadMemStats(stat2)
	t.Log(stat2.HeapInuse, stat1.HeapInuse, stat2.HeapInuse-stat1.HeapInuse, "B")
	t.Log(len(m))
}

func TestZero_Map(t *testing.T) {
	var stat1 = &runtime.MemStats{}
	runtime.ReadMemStats(stat1)

	var m = map[int]Zero{}
	for i := 0; i < 1_000_000; i++ {
		m[i] = New()
	}

	var stat2 = &runtime.MemStats{}
	runtime.ReadMemStats(stat2)
	t.Log((stat2.HeapInuse-stat1.HeapInuse)/1024/1024, "MB")
	t.Log(len(m))

	_, ok := m[1024]
	t.Log(ok)
}
