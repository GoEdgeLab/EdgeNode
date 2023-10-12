package ttlcache

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils/fasttime"
	"github.com/TeaOSLab/EdgeNode/internal/utils/testutils"
	"github.com/iwind/TeaGo/assert"
	"github.com/iwind/TeaGo/rands"
	"github.com/iwind/TeaGo/types"
	timeutil "github.com/iwind/TeaGo/utils/time"
	"runtime"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewCache(t *testing.T) {
	var cache = NewCache[int]()
	cache.Write("a", 1, time.Now().Unix()+3600)
	cache.Write("b", 2, time.Now().Unix()+1)
	cache.Write("c", 1, time.Now().Unix()+3602)
	cache.Write("d", 1, time.Now().Unix()+1)

	for _, piece := range cache.pieces {
		if len(piece.m) > 0 {
			for k, item := range piece.m {
				t.Log(k, "=>", item.Value, item.expiredAt)
			}
		}
	}
	t.Log("a:", cache.Read("a"))
	if testutils.IsSingleTesting() {
		time.Sleep(5 * time.Second)
	}

	for i := 0; i < len(cache.pieces); i++ {
		cache.GC()
	}

	t.Log("b:", cache.Read("b"))
	t.Log("d:", cache.Read("d"))
	t.Log("left:", cache.Count(), "items")
}

func TestCache_Memory(t *testing.T) {
	if !testutils.IsSingleTesting() {
		return
	}

	var cache = NewCache[int]()

	testutils.StartMemoryStats(t, func() {
		t.Log(cache.Count(), "items")
	})

	var count = 20_000_000
	for i := 0; i < count; i++ {
		cache.Write("a"+strconv.Itoa(i), 1, time.Now().Unix()+int64(rands.Int(0, 300)))
	}

	t.Log(cache.Count())

	time.Sleep(10 * time.Second)
	for i := 0; i < count; i++ {
		if i%2 == 0 {
			cache.Delete("a" + strconv.Itoa(i))
		}
	}

	t.Log(cache.Count())

	cache.Count()

	time.Sleep(3600 * time.Second)
}

func TestCache_IncreaseInt64(t *testing.T) {
	var a = assert.NewAssertion(t)

	var cache = NewCache[int64]()
	var unixTime = time.Now().Unix()

	{
		cache.IncreaseInt64("a", 1, unixTime+3600, false)
		var item = cache.Read("a")
		t.Log(item)
		a.IsTrue(item.Value == 1)
		a.IsTrue(item.expiredAt == unixTime+3600)
	}
	{
		cache.IncreaseInt64("a", 1, unixTime+3600+1, true)
		var item = cache.Read("a")
		t.Log(item)
		a.IsTrue(item.Value == 2)
		a.IsTrue(item.expiredAt == unixTime+3600+1)
	}
	{
		cache.Write("b", 1, time.Now().Unix()+3600+2)
		t.Log(cache.Read("b"))
	}
	{
		cache.IncreaseInt64("b", 1, time.Now().Unix()+3600+3, false)
		t.Log(cache.Read("b"))
	}
}

func TestCache_Read(t *testing.T) {
	runtime.GOMAXPROCS(1)

	var cache = NewCache[int](PiecesOption{Count: 32})

	for i := 0; i < 10_000_000; i++ {
		cache.Write("HELLO_WORLD_"+strconv.Itoa(i), i, time.Now().Unix()+int64(i%10240)+1)
	}
	time.Sleep(10 * time.Second)

	total := 0
	for _, piece := range cache.pieces {
		//t.Log(len(piece.m), "keys")
		total += len(piece.m)
	}
	t.Log(total, "total keys")

	before := time.Now()
	for i := 0; i < 10_240; i++ {
		_ = cache.Read("HELLO_WORLD_" + strconv.Itoa(i))
	}
	t.Log(time.Since(before).Seconds()*1000, "ms")
}

func TestCache_GC(t *testing.T) {
	if !testutils.IsSingleTesting() {
		return
	}

	var cache = NewCache[int](&PiecesOption{Count: 5})
	cache.Write("a", 1, time.Now().Unix()+1)
	cache.Write("b", 2, time.Now().Unix()+2)
	cache.Write("c", 3, time.Now().Unix()+3)
	cache.Write("d", 4, time.Now().Unix()+4)
	cache.Write("e", 5, time.Now().Unix()+10)

	go func() {
		for i := 0; i < 1000; i++ {
			cache.Write("f", 1, time.Now().Unix()+1)
			time.Sleep(10 * time.Millisecond)
		}
	}()

	for i := 0; i < 20; i++ {
		cache.GC()
		t.Log("items:", cache.Count())

		if cache.Count() == 0 {
			break
		}
		time.Sleep(1 * time.Second)
	}

	t.Log("now:", time.Now().Unix())
	for _, p := range cache.pieces {
		t.Log("expire list:", p.expiresList.Count(), p.expiresList)
		for k, v := range p.m {
			t.Log(k, v.Value, v.expiredAt)
		}
	}
}

func TestCache_GC2(t *testing.T) {
	if !testutils.IsSingleTesting() {
		return
	}

	runtime.GOMAXPROCS(1)

	var cache1 = NewCache[int](NewPiecesOption(256))
	for i := 0; i < 10_000_000; i++ {
		cache1.Write(strconv.Itoa(i), i, time.Now().Unix()+10)
	}

	var cache2 = NewCache[int](NewPiecesOption(5))
	for i := 0; i < 1_000_000; i++ {
		cache2.Write(strconv.Itoa(i), i, time.Now().Unix()+int64(rands.Int(0, 10)))
	}

	for i := 0; i < 3600; i++ {
		t.Log(timeutil.Format("H:i:s"), cache1.Count(), "items", cache2.Count(), "items")
		time.Sleep(1 * time.Second)
	}
}

func TestCacheDestroy(t *testing.T) {
	var cache = NewCache[int]()
	t.Log("count:", SharedManager.Count())
	cache.Destroy()
	t.Log("count:", SharedManager.Count())
}

func BenchmarkNewCache(b *testing.B) {
	runtime.GOMAXPROCS(1)

	var cache = NewCache[int](NewPiecesOption(128))
	for i := 0; i < 2_000_000; i++ {
		cache.Write(strconv.Itoa(i), i, time.Now().Unix()+int64(rands.Int(10, 100)))
	}
	b.Log("start reading ...")

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cache.Read(strconv.Itoa(rands.Int(0, 999999)))
		}
	})
}

func BenchmarkCache_Add(b *testing.B) {
	runtime.GOMAXPROCS(1)

	var cache = NewCache[int]()
	for i := 0; i < b.N; i++ {
		cache.Write(strconv.Itoa(i), i, fasttime.Now().Unix()+int64(i%1024))
	}
}

func BenchmarkCache_Add_Parallel(b *testing.B) {
	runtime.GOMAXPROCS(1)

	var cache = NewCache[int64]()
	var i int64
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var j = atomic.AddInt64(&i, 1)
			cache.Write(types.String(j%1e6), j, fasttime.Now().Unix()+i%1024)
		}
	})
}

func BenchmarkNewCacheGC(b *testing.B) {
	runtime.GOMAXPROCS(1)

	var cache = NewCache[int](NewPiecesOption(1024))
	for i := 0; i < 3_000_000; i++ {
		cache.Write(strconv.Itoa(i), i, time.Now().Unix()+int64(rands.Int(0, 100)))
	}
	//b.Log(cache.pieces[0].Count())

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cache.GC()
		}
	})
}

func BenchmarkNewCacheClean(b *testing.B) {
	runtime.GOMAXPROCS(1)

	var cache = NewCache[int](NewPiecesOption(128))
	for i := 0; i < 3_000_000; i++ {
		cache.Write(strconv.Itoa(i), i, time.Now().Unix()+int64(rands.Int(10, 100)))
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cache.Clean()
		}
	})
}
