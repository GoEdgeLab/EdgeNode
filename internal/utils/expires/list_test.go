package expires_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils/expires"
	"github.com/TeaOSLab/EdgeNode/internal/utils/fasttime"
	"github.com/TeaOSLab/EdgeNode/internal/utils/testutils"
	"github.com/iwind/TeaGo/assert"
	"github.com/iwind/TeaGo/logs"
	timeutil "github.com/iwind/TeaGo/utils/time"
	"math"
	"math/rand"
	"runtime"
	"testing"
	"time"
)

func TestList_Add(t *testing.T) {
	var list = expires.NewList()
	list.Add(1, time.Now().Unix())
	t.Log("===BEFORE===")
	logs.PrintAsJSON(list.ExpireMap(), t)
	logs.PrintAsJSON(list.ItemsMap(), t)

	list.Add(1, time.Now().Unix()+1)
	list.Add(2, time.Now().Unix()+1)
	list.Add(3, time.Now().Unix()+2)
	t.Log("===AFTER===")
	logs.PrintAsJSON(list.ExpireMap(), t)
	logs.PrintAsJSON(list.ItemsMap(), t)
}

func TestList_Add_Overwrite(t *testing.T) {
	var timestamp = time.Now().Unix()

	var list = expires.NewList()
	list.Add(1, timestamp+1)
	list.Add(1, timestamp+1)
	list.Add(2, timestamp+1)
	list.Add(1, timestamp+2)
	logs.PrintAsJSON(list.ExpireMap(), t)
	logs.PrintAsJSON(list.ItemsMap(), t)

	var a = assert.NewAssertion(t)
	a.IsTrue(len(list.ItemsMap()) == 2)
	a.IsTrue(len(list.ExpireMap()) == 2)
	a.IsTrue(list.ItemsMap()[1] == timestamp+2)
}

func TestList_Remove(t *testing.T) {
	var a = assert.NewAssertion(t)

	var list = expires.NewList()
	list.Add(1, time.Now().Unix()+1)
	list.Remove(1)
	logs.PrintAsJSON(list.ExpireMap(), t)
	logs.PrintAsJSON(list.ItemsMap(), t)

	a.IsTrue(len(list.ExpireMap()) == 0)
	a.IsTrue(len(list.ItemsMap()) == 0)
}

func TestList_GC(t *testing.T) {
	var unixTime = time.Now().Unix()
	t.Log("unixTime:", unixTime)

	var list = expires.NewList()
	list.Add(1, unixTime+1)
	list.Add(2, unixTime+1)
	list.Add(3, unixTime+2)
	list.OnGC(func(itemId uint64) {
		t.Log("gc:", itemId)
	})
	t.Log("last unixTime:", list.LastTimestamp())
	list.GC(time.Now().Unix() + 2)
	logs.PrintAsJSON(list.ExpireMap(), t)
	logs.PrintAsJSON(list.ItemsMap(), t)

	t.Log(list.Count())
}

func TestList_GC_Batch(t *testing.T) {
	var list = expires.NewList()
	list.Add(1, time.Now().Unix()+1)
	list.Add(2, time.Now().Unix()+1)
	list.Add(3, time.Now().Unix()+2)
	list.Add(4, time.Now().Unix()+2)
	list.OnGCBatch(func(itemMap expires.ItemMap) {
		t.Log("gc:", itemMap)
	})
	list.GC(time.Now().Unix() + 2)
	logs.PrintAsJSON(list.ExpireMap(), t)
	logs.PrintAsJSON(list.ItemsMap(), t)
}

func TestList_Start_GC(t *testing.T) {
	if !testutils.IsSingleTesting() {
		return
	}

	var list = expires.NewList()
	list.Add(1, time.Now().Unix()+1)
	list.Add(2, time.Now().Unix()+1)
	list.Add(3, time.Now().Unix()+2)
	list.Add(3, time.Now().Unix()+10)
	list.Add(4, time.Now().Unix()+5)
	list.Add(5, time.Now().Unix()+5)
	list.Add(6, time.Now().Unix()+6)
	list.Add(7, time.Now().Unix()+6)
	list.Add(8, time.Now().Unix()+6)

	list.OnGC(func(itemId uint64) {
		t.Log("gc:", itemId, timeutil.Format("H:i:s"))
		time.Sleep(2 * time.Second)
	})

	go func() {
		expires.SharedManager.Add(list)
	}()

	time.Sleep(20 * time.Second)
	logs.PrintAsJSON(list.ItemsMap())
	logs.PrintAsJSON(list.ExpireMap())
}

func TestList_ManyItems(t *testing.T) {
	var list = expires.NewList()
	for i := 0; i < 1_000; i++ {
		list.Add(uint64(i), time.Now().Unix())
	}
	for i := 0; i < 1_000; i++ {
		list.Add(uint64(i), time.Now().Unix()+1)
	}

	var now = time.Now()
	var count = 0
	list.OnGC(func(itemId uint64) {
		count++
	})
	list.GC(time.Now().Unix() + 1)
	t.Log("gc", count, "items")
	t.Log(time.Since(now))
}

func TestList_Memory(t *testing.T) {
	if !testutils.IsSingleTesting() {
		return
	}

	var list = expires.NewList()

	testutils.StartMemoryStats(t, func() {
		t.Log(list.Count(), "items")
	})

	for i := 0; i < 10_000_000; i++ {
		list.Add(uint64(i), time.Now().Unix()+1800)
	}

	time.Sleep(1 * time.Hour)
}

func TestList_Map_Performance(t *testing.T) {
	t.Log("max uint32", math.MaxUint32)

	var timestamp = time.Now().Unix()

	{
		var m = map[int64]int64{}
		for i := 0; i < 1_000_000; i++ {
			m[int64(i)] = timestamp
		}

		var now = time.Now()
		for i := 0; i < 100_000; i++ {
			delete(m, int64(i))
		}
		t.Log(time.Since(now))
	}

	{
		var m = map[uint64]int64{}
		for i := 0; i < 1_000_000; i++ {
			m[uint64(i)] = timestamp
		}

		var now = time.Now()
		for i := 0; i < 100_000; i++ {
			delete(m, uint64(i))
		}
		t.Log(time.Since(now))
	}

	{
		var m = map[uint32]int64{}
		for i := 0; i < 1_000_000; i++ {
			m[uint32(i)] = timestamp
		}

		var now = time.Now()
		for i := 0; i < 100_000; i++ {
			delete(m, uint32(i))
		}
		t.Log(time.Since(now))
	}
}

func BenchmarkList_Add(b *testing.B) {
	var list = expires.NewList()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			list.Add(rand.Uint64(), fasttime.Now().Unix()+int64(rand.Int()%10_000_000))
		}
	})
}

func Benchmark_Map_Uint64(b *testing.B) {
	runtime.GOMAXPROCS(1)
	var timestamp = uint64(time.Now().Unix())

	var i uint64
	var count uint64 = 1_000_000

	var m = map[uint64]uint64{}
	for i = 0; i < count; i++ {
		m[i] = timestamp
	}

	for n := 0; n < b.N; n++ {
		for i = 0; i < count; i++ {
			_ = m[i]
		}
	}
}

func BenchmarkList_GC(b *testing.B) {
	runtime.GOMAXPROCS(4)

	var lists = []*expires.List{}

	for m := 0; m < 1_000; m++ {
		var list = expires.NewList()
		for j := 0; j < 10_000; j++ {
			list.Add(uint64(j), fasttime.Now().Unix()+int64(rand.Int()%10_000_000))
		}
		lists = append(lists, list)
	}

	var timestamp = time.Now().Unix()

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			for _, list := range lists {
				list.GC(timestamp + int64(rand.Int()%1_000_000))
			}
		}
	})
}
