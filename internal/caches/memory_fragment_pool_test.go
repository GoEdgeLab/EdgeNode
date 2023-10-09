// Copyright 2023 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package caches_test

import (
	"bytes"
	"github.com/TeaOSLab/EdgeNode/internal/caches"
	"github.com/TeaOSLab/EdgeNode/internal/utils/testutils"
	"github.com/iwind/TeaGo/assert"
	"github.com/iwind/TeaGo/rands"
	timeutil "github.com/iwind/TeaGo/utils/time"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewMemoryFragmentPool(t *testing.T) {
	var a = assert.NewAssertion(t)

	var pool = caches.NewMemoryFragmentPool()
	for i := 0; i < 3000; i++ {
		ok := pool.Put(make([]byte, 2<<20))
		if !ok {
			t.Log("finished at", i)
			break
		}
	}

	t.Log(pool.TotalSize()>>20, "MB", pool.Len(), "items")

	{
		r, ok := pool.Get(1 << 20)
		a.IsTrue(ok)
		a.IsTrue(len(r) == 1<<20)
	}

	{
		r, ok := pool.Get(2 << 20)
		a.IsTrue(ok)
		a.IsTrue(len(r) == 2<<20)
	}

	{
		r, ok := pool.Get(4 << 20)
		a.IsFalse(ok)
		a.IsTrue(len(r) == 0)
	}

	t.Log(pool.TotalSize()>>20, "MB", pool.Len(), "items")
}

func TestNewMemoryFragmentPool_LargeBucket(t *testing.T) {
	var a = assert.NewAssertion(t)

	var pool = caches.NewMemoryFragmentPool()
	{
		pool.Put(make([]byte, 128<<20+1))
		a.IsTrue(pool.Len() == 0)
	}

	{
		pool.Put(make([]byte, 128<<20))
		a.IsTrue(pool.Len() == 1)

		pool.Get(118 << 20)
		a.IsTrue(pool.Len() == 0)
	}

	{
		pool.Put(make([]byte, 128<<20))
		a.IsTrue(pool.Len() == 1)

		pool.Get(110 << 20)
		a.IsTrue(pool.Len() == 1)
	}
}

func TestMemoryFragmentPool_Get_Exactly(t *testing.T) {
	var a = assert.NewAssertion(t)

	var pool = caches.NewMemoryFragmentPool()
	{
		pool.Put(make([]byte, 129<<20))
		a.IsTrue(pool.Len() == 0)
	}

	{
		pool.Put(make([]byte, 4<<20))
		a.IsTrue(pool.Len() == 1)
	}

	{
		pool.Get(4 << 20)
		a.IsTrue(pool.Len() == 0)
	}
}

func TestMemoryFragmentPool_Get_Round(t *testing.T) {
	var a = assert.NewAssertion(t)

	var pool = caches.NewMemoryFragmentPool()
	{
		pool.Put(make([]byte, 8<<20))
		pool.Put(make([]byte, 8<<20))
		pool.Put(make([]byte, 8<<20))
		a.IsTrue(pool.Len() == 3)
	}

	{
		resultBytes, ok := pool.Get(3 << 20)
		a.IsTrue(pool.Len() == 2)
		if ok {
			pool.Put(resultBytes)
		}
	}

	{
		pool.Get(2 << 20)
		a.IsTrue(pool.Len() == 2)
	}

	{
		pool.Get(1 << 20)
		a.IsTrue(pool.Len() == 1)
	}
}

func TestMemoryFragmentPool_GC(t *testing.T) {
	var pool = caches.NewMemoryFragmentPool()
	pool.SetCapacity(32 << 20)
	for i := 0; i < 16; i++ {
		pool.Put(make([]byte, 4<<20))
	}
	var before = time.Now()
	pool.GC()
	t.Log(time.Since(before).Seconds()*1000, "ms")
	t.Log(pool.Len())
}

func TestMemoryFragmentPool_Memory(t *testing.T) {
	if !testutils.IsSingleTesting() {
		return
	}

	var pool = caches.NewMemoryFragmentPool()

	testutils.StartMemoryStats(t, func() {
		t.Log(pool.Len(), "items")
	})

	var sampleData = bytes.Repeat([]byte{'A'}, 16<<20)

	var countNew = 0
	for i := 0; i < 1000; i++ {
		cacheData, ok := pool.Get(16 << 20)
		if ok {
			copy(cacheData, sampleData)
			pool.Put(cacheData)
		} else {
			countNew++
			var data = make([]byte, 16<<20)
			copy(data, sampleData)
			pool.Put(data)
		}
	}

	t.Log("count new:", countNew)
	t.Log("count remains:", pool.Len())

	time.Sleep(10 * time.Minute)
}

func TestMemoryFragmentPool_GCNextBucket(t *testing.T) {
	if !testutils.IsSingleTesting() {
		return
	}

	var pool = caches.NewMemoryFragmentPool()
	for i := 0; i < 1000; i++ {
		pool.Put(make([]byte, rands.Int(0, 100)<<20))
	}

	var lastLen int
	for {
		pool.GCNextBucket()
		var currentLen = pool.Len()
		if lastLen == currentLen {
			continue
		}
		lastLen = currentLen

		t.Log(currentLen, "items", pool.TotalSize(), "bytes", timeutil.Format("H:i:s"))
		time.Sleep(100 * time.Millisecond)

		if currentLen == 0 {
			break
		}
	}
}

func TestMemoryFragmentPoolItem(t *testing.T) {
	var a = assert.NewAssertion(t)

	var m = map[int]*caches.MemoryFragmentPoolItem{}
	m[1] = &caches.MemoryFragmentPoolItem{
		Refs: 0,
	}
	var item = m[1]
	a.IsTrue(item.Refs == 0)
	a.IsTrue(atomic.AddInt32(&item.Refs, 1) == 1)

	for _, item2 := range m {
		t.Log(item2)
		a.IsTrue(atomic.AddInt32(&item2.Refs, 1) == 2)
	}

	t.Log(m)
}

func BenchmarkMemoryFragmentPool_Get_HIT(b *testing.B) {
	runtime.GOMAXPROCS(4)

	var pool = caches.NewMemoryFragmentPool()
	for i := 0; i < 3000; i++ {
		ok := pool.Put(make([]byte, 2<<20))
		if !ok {
			break
		}
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			data, ok := pool.Get(2 << 20)
			if ok {
				pool.Put(data)
			}
		}
	})
}

func BenchmarkMemoryFragmentPool_Get_TOTALLY_MISSING(b *testing.B) {
	runtime.GOMAXPROCS(4)

	var pool = caches.NewMemoryFragmentPool()
	for i := 0; i < 3000; i++ {
		ok := pool.Put(make([]byte, 2<<20+100))
		if !ok {
			break
		}
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			data, ok := pool.Get(2<<20 + 200)
			if ok {
				pool.Put(data)
			}
		}
	})
}

func BenchmarkMemoryPool_Get_HIT_MISSING(b *testing.B) {
	runtime.GOMAXPROCS(4)

	var pool = caches.NewMemoryFragmentPool()
	for i := 0; i < 3000; i++ {
		ok := pool.Put(make([]byte, rands.Int(2, 32)<<20))
		if !ok {
			break
		}
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			data, ok := pool.Get(4 << 20)
			if ok {
				pool.Put(data)
			}
		}
	})
}

func BenchmarkMemoryFragmentPool_GC(b *testing.B) {
	runtime.GOMAXPROCS(4)

	var pool = caches.NewMemoryFragmentPool()
	pool.SetCapacity(1 << 30)
	for i := 0; i < 2_000; i++ {
		pool.Put(make([]byte, 1<<20))
	}

	var mu = sync.Mutex{}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mu.Lock()
			for i := 0; i < 100; i++ {
				pool.GCNextBucket()
			}
			mu.Unlock()
		}
	})
}
