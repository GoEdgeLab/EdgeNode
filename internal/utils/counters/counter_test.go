// Copyright 2023 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package counters_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils/counters"
	"github.com/TeaOSLab/EdgeNode/internal/utils/testutils"
	"github.com/iwind/TeaGo/assert"
	"github.com/iwind/TeaGo/rands"
	"github.com/iwind/TeaGo/types"
	timeutil "github.com/iwind/TeaGo/utils/time"
	"runtime"
	"runtime/debug"
	"sync/atomic"
	"testing"
	"time"
)

func TestCounter_Increase(t *testing.T) {
	var a = assert.NewAssertion(t)

	var counter = counters.NewCounter[uint32]()
	a.IsTrue(counter.Increase(1, 10) == 1)
	a.IsTrue(counter.Increase(1, 10) == 2)
	a.IsTrue(counter.Increase(2, 10) == 1)

	counter.Reset(1)
	a.IsTrue(counter.Get(1) == 0) // changed
	a.IsTrue(counter.Get(2) == 1) // not changed
}

func TestCounter_IncreaseKey(t *testing.T) {
	var a = assert.NewAssertion(t)

	var counter = counters.NewCounter[uint32]()
	a.IsTrue(counter.IncreaseKey("1", 10) == 1)
	a.IsTrue(counter.IncreaseKey("1", 10) == 2)
	a.IsTrue(counter.IncreaseKey("2", 10) == 1)

	counter.ResetKey("1")
	a.IsTrue(counter.GetKey("1") == 0) // changed
	a.IsTrue(counter.GetKey("2") == 1) // not changed
}

func TestCounter_GC(t *testing.T) {
	if !testutils.IsSingleTesting() {
		return
	}

	var counter = counters.NewCounter[uint32]()
	counter.Increase(1, 20)
	time.Sleep(1 * time.Second)
	counter.Increase(1, 20)
	time.Sleep(1 * time.Second)
	counter.Increase(1, 20)
	counter.GC()
	t.Log(counter.Get(1))
}

func TestCounter_GC2(t *testing.T) {
	if !testutils.IsSingleTesting() {
		return
	}

	var counter = counters.NewCounter[uint32]().WithGC()
	for i := 0; i < 100_000; i++ {
		counter.Increase(uint64(i), rands.Int(10, 300))
	}

	var ticker = time.NewTicker(1 * time.Second)
	for range ticker.C {
		t.Log(timeutil.Format("H:i:s"), counter.TotalItems())
		if counter.TotalItems() == 0 {
			break
		}
	}
}

func TestCounterMemory(t *testing.T) {
	var stat = &runtime.MemStats{}
	runtime.ReadMemStats(stat)

	var counter = counters.NewCounter[uint32]()
	for i := 0; i < 1_000_000; i++ {
		counter.Increase(uint64(i), rands.Int(10, 300))
	}

	runtime.GC()
	runtime.GC()
	debug.FreeOSMemory()

	var stat1 = &runtime.MemStats{}
	runtime.ReadMemStats(stat1)
	t.Log((stat1.HeapInuse-stat.HeapInuse)/(1<<20), "MB")

	t.Log(counter.TotalItems())

	var gcPause = func() {
		var before = time.Now()
		runtime.GC()
		var costSeconds = time.Since(before).Seconds()
		var stats = &debug.GCStats{}
		debug.ReadGCStats(stats)
		t.Log("GC pause:", stats.PauseTotal.Seconds()*1000, "ms", "cost:", costSeconds*1000, "ms")
	}

	gcPause()

	_ = counter.TotalItems()
}

func BenchmarkCounter_Increase(b *testing.B) {
	runtime.GOMAXPROCS(4)

	var counter = counters.NewCounter[uint32]()
	b.ResetTimer()

	var i uint64
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			counter.Increase(atomic.AddUint64(&i, 1)%1e6, 20)
		}
	})

	//b.Log(counter.TotalItems())
}

func BenchmarkCounter_IncreaseKey(b *testing.B) {
	runtime.GOMAXPROCS(4)

	var counter = counters.NewCounter[uint32]()

	go func() {
		var ticker = time.NewTicker(100 * time.Millisecond)
		for range ticker.C {
			counter.GC()
		}
	}()

	b.ResetTimer()

	var i uint64
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			counter.IncreaseKey(types.String(atomic.AddUint64(&i, 1)%1e6), 20)
		}
	})

	//b.Log(counter.TotalItems())
}

func BenchmarkCounter_IncreaseKey2(b *testing.B) {
	runtime.GOMAXPROCS(4)

	var counter = counters.NewCounter[uint32]()

	go func() {
		var ticker = time.NewTicker(1 * time.Millisecond)
		for range ticker.C {
			counter.GC()
		}
	}()

	b.ResetTimer()

	var i uint64
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			counter.IncreaseKey(types.String(atomic.AddUint64(&i, 1)%1e5), 20)
		}
	})

	//b.Log(counter.TotalItems())
}

func BenchmarkCounter_GC(b *testing.B) {
	runtime.GOMAXPROCS(4)

	var counter = counters.NewCounter[uint32]()

	for i := uint64(0); i < 1e5; i++ {
		counter.IncreaseKey(types.String(i), 20)
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			counter.GC()
		}
	})

	//b.Log(counter.TotalItems())
}
