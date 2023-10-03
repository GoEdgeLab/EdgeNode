// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package linkedlist_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils/linkedlist"
	"runtime"
	"strconv"
	"testing"
)

func TestNewList_Memory(t *testing.T) {
	var stat1 = &runtime.MemStats{}
	runtime.ReadMemStats(stat1)

	var list = linkedlist.NewList[int]()
	for i := 0; i < 1_000_000; i++ {
		var item = &linkedlist.Item[int]{}
		list.Push(item)
	}

	runtime.GC()

	var stat2 = &runtime.MemStats{}
	runtime.ReadMemStats(stat2)
	t.Log((stat2.HeapInuse-stat1.HeapInuse)>>20, "MB")
	t.Log(list.Len())

	var count = 0
	list.Range(func(item *linkedlist.Item[int]) (goNext bool) {
		count++
		return true
	})
	t.Log(count)
}

func TestNewList_Memory_String(t *testing.T) {
	var stat1 = &runtime.MemStats{}
	runtime.ReadMemStats(stat1)

	var list = linkedlist.NewList[string]()
	for i := 0; i < 1_000_000; i++ {
		var item = &linkedlist.Item[string]{}
		item.Value = strconv.Itoa(i)
		list.Push(item)
	}

	runtime.GC()

	var stat2 = &runtime.MemStats{}
	runtime.ReadMemStats(stat2)
	t.Log((stat2.HeapInuse-stat1.HeapInuse)>>20, "MB")
	t.Log(list.Len())
}

func TestList_Push(t *testing.T) {
	var list = linkedlist.NewList[int]()
	list.Push(linkedlist.NewItem(1))
	list.Push(linkedlist.NewItem(2))

	var item3 = linkedlist.NewItem(3)
	list.Push(item3)

	var item4 = linkedlist.NewItem(4)
	list.Push(item4)
	list.Range(func(item *linkedlist.Item[int]) (goNext bool) {
		t.Log(item.Value)
		return true
	})

	t.Log("=== after push 3 ===")
	list.Push(item3)
	list.Range(func(item *linkedlist.Item[int]) (goNext bool) {
		t.Log(item.Value)
		return true
	})

	t.Log("=== after push 4 ===")
	list.Push(item4)
	list.Push(item3)
	list.Push(item3)
	list.Push(item3)
	list.Push(item4)
	list.Push(item4)
	list.Range(func(item *linkedlist.Item[int]) (goNext bool) {
		t.Log(item.Value)
		return true
	})

	t.Log("=== after remove 3 ===")
	list.Remove(item3)
	list.Range(func(item *linkedlist.Item[int]) (goNext bool) {
		t.Log(item.Value)
		return true
	})
}

func BenchmarkList_Add(b *testing.B) {
	var list = linkedlist.NewList[int]()
	for i := 0; i < b.N; i++ {
		var item = &linkedlist.Item[int]{}
		list.Push(item)
	}
}
