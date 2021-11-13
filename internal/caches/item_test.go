// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package caches

import (
	"github.com/iwind/TeaGo/rands"
	"github.com/iwind/TeaGo/types"
	"runtime"
	"testing"
	"time"
)

func TestItem_IncreaseHit(t *testing.T) {
	var week = currentWeek()

	var item = &Item{}
	//item.Week = 2704
	item.Week2Hits = 100
	item.IncreaseHit(week)
	t.Log("week:", item.Week, "week1:", item.Week1Hits, "week2:", item.Week2Hits)

	item.IncreaseHit(week)
	t.Log("week:", item.Week, "week1:", item.Week1Hits, "week2:", item.Week2Hits)
}

func TestItems_Memory(t *testing.T) {
	var stat = &runtime.MemStats{}
	runtime.ReadMemStats(stat)
	var memory1 = stat.HeapInuse

	var items = []*Item{}
	for i := 0; i < 10_000_000; i++ {
		items = append(items, &Item{
			Key: types.String(i),
		})
	}

	runtime.ReadMemStats(stat)
	var memory2 = stat.HeapInuse

	t.Log(memory1, memory2, (memory2-memory1)/1024/1024, "M")

	var weekItems = make(map[string]*Item, 10_000_000)

	for _, item := range items {
		weekItems[item.Key] = item
	}

	runtime.ReadMemStats(stat)
	var memory3 = stat.HeapInuse
	t.Log(memory2, memory3, (memory3-memory2)/1024/1024, "M")

	time.Sleep(1 * time.Second)
	t.Log(len(items), len(weekItems))
}

func TestItems_Memory2(t *testing.T) {
	var stat = &runtime.MemStats{}
	runtime.ReadMemStats(stat)
	var memory1 = stat.HeapInuse

	var items = map[int32]map[string]bool{}
	for i := 0; i < 10_000_000; i++ {
		var week = int32((time.Now().Unix() - int64(86400*rands.Int(0, 300))) / (86400 * 7))
		m, ok := items[week]
		if !ok {
			m = map[string]bool{}
			items[week] = m
		}
		m[types.String(int64(i)*1_000_000)] = true
	}

	runtime.ReadMemStats(stat)
	var memory2 = stat.HeapInuse

	t.Log(memory1, memory2, (memory2-memory1)/1024/1024, "M")

	time.Sleep(1 * time.Second)
	for w, i := range items {
		t.Log(w, len(i))
	}
}
