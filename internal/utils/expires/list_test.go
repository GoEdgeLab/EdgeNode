package expires

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils/fasttime"
	"github.com/TeaOSLab/EdgeNode/internal/utils/testutils"
	"github.com/iwind/TeaGo/assert"
	"github.com/iwind/TeaGo/logs"
	timeutil "github.com/iwind/TeaGo/utils/time"
	"math"
	"runtime"
	"testing"
	"time"
)

func TestList_Add(t *testing.T) {
	list := NewList()
	list.Add(1, time.Now().Unix())
	t.Log("===BEFORE===")
	logs.PrintAsJSON(list.expireMap, t)
	logs.PrintAsJSON(list.itemsMap, t)

	list.Add(1, time.Now().Unix()+1)
	list.Add(2, time.Now().Unix()+1)
	list.Add(3, time.Now().Unix()+2)
	t.Log("===AFTER===")
	logs.PrintAsJSON(list.expireMap, t)
	logs.PrintAsJSON(list.itemsMap, t)
}

func TestList_Add_Overwrite(t *testing.T) {
	var timestamp = time.Now().Unix()

	list := NewList()
	list.Add(1, timestamp+1)
	list.Add(1, timestamp+1)
	list.Add(1, timestamp+2)
	logs.PrintAsJSON(list.expireMap, t)
	logs.PrintAsJSON(list.itemsMap, t)

	var a = assert.NewAssertion(t)
	a.IsTrue(len(list.itemsMap) == 1)
	a.IsTrue(len(list.expireMap) == 1)
	a.IsTrue(list.itemsMap[1] == timestamp+2)
}

func TestList_Remove(t *testing.T) {
	list := NewList()
	list.Add(1, time.Now().Unix()+1)
	list.Remove(1)
	logs.PrintAsJSON(list.expireMap, t)
	logs.PrintAsJSON(list.itemsMap, t)
}

func TestList_GC(t *testing.T) {
	var unixTime = time.Now().Unix()
	t.Log("unixTime:", unixTime)

	var list = NewList()
	list.Add(1, unixTime+1)
	list.Add(2, unixTime+1)
	list.Add(3, unixTime+2)
	list.OnGC(func(itemId uint64) {
		t.Log("gc:", itemId)
	})
	t.Log("last unixTime:", list.lastTimestamp)
	list.GC(time.Now().Unix() + 2)
	logs.PrintAsJSON(list.expireMap, t)
	logs.PrintAsJSON(list.itemsMap, t)

	t.Log(list.Count())
}

func TestList_GC_Batch(t *testing.T) {
	list := NewList()
	list.Add(1, time.Now().Unix()+1)
	list.Add(2, time.Now().Unix()+1)
	list.Add(3, time.Now().Unix()+2)
	list.Add(4, time.Now().Unix()+2)
	list.OnGCBatch(func(itemMap ItemMap) {
		t.Log("gc:", itemMap)
	})
	list.GC(time.Now().Unix() + 2)
	logs.PrintAsJSON(list.expireMap, t)
	logs.PrintAsJSON(list.itemsMap, t)
}

func TestList_Start_GC(t *testing.T) {
	if !testutils.IsSingleTesting() {
		return
	}

	list := NewList()
	list.Add(1, time.Now().Unix()+1)
	list.Add(2, time.Now().Unix()+1)
	list.Add(3, time.Now().Unix()+2)
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
		SharedManager.Add(list)
	}()

	time.Sleep(20 * time.Second)
}

func TestList_ManyItems(t *testing.T) {
	list := NewList()
	for i := 0; i < 1_000; i++ {
		list.Add(uint64(i), time.Now().Unix())
	}
	for i := 0; i < 1_000; i++ {
		list.Add(uint64(i), time.Now().Unix()+1)
	}

	now := time.Now()
	count := 0
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

	var list = NewList()

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
		m := map[int64]int64{}
		for i := 0; i < 1_000_000; i++ {
			m[int64(i)] = timestamp
		}

		now := time.Now()
		for i := 0; i < 100_000; i++ {
			delete(m, int64(i))
		}
		t.Log(time.Since(now))
	}

	{
		m := map[uint64]int64{}
		for i := 0; i < 1_000_000; i++ {
			m[uint64(i)] = timestamp
		}

		now := time.Now()
		for i := 0; i < 100_000; i++ {
			delete(m, uint64(i))
		}
		t.Log(time.Since(now))
	}

	{
		m := map[uint32]int64{}
		for i := 0; i < 1_000_000; i++ {
			m[uint32(i)] = timestamp
		}

		now := time.Now()
		for i := 0; i < 100_000; i++ {
			delete(m, uint32(i))
		}
		t.Log(time.Since(now))
	}
}

func Benchmark_Map_Uint64(b *testing.B) {
	runtime.GOMAXPROCS(1)
	var timestamp = uint64(time.Now().Unix())

	var i uint64
	var count uint64 = 1_000_000

	m := map[uint64]uint64{}
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
	runtime.GOMAXPROCS(1)

	var lists = []*List{}

	for m := 0; m < 1_000; m++ {
		var list = NewList()
		for j := 0; j < 10_000; j++ {
			list.Add(uint64(j), fasttime.Now().Unix()+100)
		}
		lists = append(lists, list)
	}

	b.ResetTimer()

	var timestamp = time.Now().Unix()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			for _, list := range lists {
				list.GC(timestamp)
			}
		}
	})
}
