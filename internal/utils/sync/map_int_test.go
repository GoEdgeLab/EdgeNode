// Copyright 2023 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package syncutils_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/stats"
	syncutils "github.com/TeaOSLab/EdgeNode/internal/utils/sync"
	"github.com/iwind/TeaGo/assert"
	"github.com/iwind/TeaGo/types"
	"sync"
	"testing"
)

func TestIntMap(t *testing.T) {
	var a = assert.NewAssertion(t)

	var m = syncutils.NewIntMap[int, string]()
	m.Put(1, "1")
	a.IsTrue(m.Has(1))
	a.IsFalse(m.Has(2))
	m.Put(-1, "-1")
	t.Log(m.Get(-1))
	t.Log(m.Len(), "values")
	{
		a.IsTrue(m.Has(-1))
		m.Delete(-1)
		a.IsFalse(m.Has(-1))
	}
	t.Log(m.Len(), "values")
}

func TestInt64Map(t *testing.T) {
	var a = assert.NewAssertion(t)

	var m = syncutils.NewIntMap[int64, string]()
	m.Put(1, "1")
	a.IsTrue(m.Has(1))
	a.IsFalse(m.Has(2))
	m.Put(-1, "-1")
	t.Log(m.Get(-1))
	t.Log(m.Len(), "values")
	{
		a.IsTrue(m.Has(-1))
		m.Delete(-1)
		a.IsFalse(m.Has(-1))
	}
	m.Put(1024000000, "large int")
	t.Log(m.Get(1024000000))
	t.Log(m.Len(), "values")
}

func TestIntMap_PutCompact(t *testing.T) {
	var a = assert.NewAssertion(t)

	var m = syncutils.NewIntMap[int, string]()
	m.Put(1, "a")
	m.Put(1, "b")
	a.IsTrue(m.Get(1) == "b")

	m.PutCompact(1, "c", func(oldV string, newV string) string {
		return oldV + newV
	})

	a.IsTrue(m.Get(1) == "bc")
}

func TestIntMap_ForEachRead(t *testing.T) {
	var m = syncutils.NewIntMap[int, string]()
	for i := 0; i < 100; i++ {
		m.Put(i, "v"+types.String(i))
	}

	t.Log(m.Len())

	m.ForEachRead(func(k int, v string) {
		t.Log(k, v)
	})
}

func TestIntMap_ForEachWrite(t *testing.T) {
	var m = syncutils.NewIntMap[int, string]()
	for i := 0; i < 100; i++ {
		m.Put(i, "v"+types.String(i))
	}

	t.Log(m.Len())

	m.ForEachRead(func(k int, v string) {
		t.Log(k, v)
		m.DeleteUnsafe(k)
	})
	t.Log(m.Len(), "elements left")
}

func BenchmarkNewIntMap(b *testing.B) {
	var m = syncutils.NewIntMap[int, *stats.BandwidthStat]()

	b.RunParallel(func(pb *testing.PB) {
		var i int
		for pb.Next() {
			i++
			m.Put(i, &stats.BandwidthStat{ServerId: 100})
			_ = m.Get(i + 100)
		}
	})
}

func BenchmarkNewIntMap2(b *testing.B) {
	var m = map[int]*stats.BandwidthStat{}
	var locker = sync.RWMutex{}

	b.RunParallel(func(pb *testing.PB) {
		var i int
		for pb.Next() {
			i++
			locker.Lock()
			m[i] = &stats.BandwidthStat{ServerId: 100}
			locker.Unlock()

			locker.RLock()
			_ = m[i+100]
			locker.RUnlock()
		}
	})
}
