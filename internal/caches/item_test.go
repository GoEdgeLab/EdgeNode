// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package caches_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/caches"
	"github.com/TeaOSLab/EdgeNode/internal/zero"
	"github.com/iwind/TeaGo/rands"
	"github.com/iwind/TeaGo/types"
	"runtime"
	"testing"
	"time"
)

func TestItems_Memory(t *testing.T) {
	var stat = &runtime.MemStats{}
	runtime.ReadMemStats(stat)
	var memory1 = stat.HeapInuse

	var items = []*caches.Item{}
	for i := 0; i < 10_000_000; i++ {
		items = append(items, &caches.Item{
			Key: types.String(i),
		})
	}

	runtime.ReadMemStats(stat)
	var memory2 = stat.HeapInuse

	t.Log(memory1, memory2, (memory2-memory1)/1024/1024, "M")

	runtime.ReadMemStats(stat)
	var memory3 = stat.HeapInuse
	t.Log(memory2, memory3, (memory3-memory2)/1024/1024, "M")

	time.Sleep(1 * time.Second)
}

func TestItems_Memory2(t *testing.T) {
	var stat = &runtime.MemStats{}
	runtime.ReadMemStats(stat)
	var memory1 = stat.HeapInuse

	var items = map[int32]map[string]zero.Zero{}
	for i := 0; i < 10_000_000; i++ {
		var week = int32((time.Now().Unix() - int64(86400*rands.Int(0, 300))) / (86400 * 7))
		m, ok := items[week]
		if !ok {
			m = map[string]zero.Zero{}
			items[week] = m
		}
		m[types.String(int64(i)*1_000_000)] = zero.New()
	}

	runtime.ReadMemStats(stat)
	var memory2 = stat.HeapInuse

	t.Log(memory1, memory2, (memory2-memory1)/1024/1024, "M")

	time.Sleep(1 * time.Second)
	for w, i := range items {
		t.Log(w, len(i))
	}
}

func TestItem_RequestURI(t *testing.T) {
	for _, u := range []string{
		"https://goedge.cn/hello/world",
		"https://goedge.cn:8080/hello/world",
		"https://goedge.cn/hello/world?v=1&t=123",
	} {
		var item = &caches.Item{Key: u}
		t.Log(u, "=>", item.RequestURI())
	}
}
