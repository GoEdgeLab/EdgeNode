package utils

import (
	"github.com/iwind/TeaGo/assert"
	"runtime"
	"testing"
)

func TestNewBytePool(t *testing.T) {
	a := assert.NewAssertion(t)

	pool := NewBytePool(5, 8)
	buf := pool.Get()
	a.IsTrue(len(buf) == 8)
	a.IsTrue(len(pool.c) == 0)

	pool.Put(buf)
	a.IsTrue(len(pool.c) == 1)

	pool.Get()
	a.IsTrue(len(pool.c) == 0)

	for i := 0; i < 10; i++ {
		pool.Put(buf)
	}
	t.Log(len(pool.c))
	a.IsTrue(len(pool.c) == 5)
}

func TestBytePool_Memory(t *testing.T) {
	var stat1 = &runtime.MemStats{}
	runtime.ReadMemStats(stat1)

	var pool = NewBytePool(20480, 32*1024)
	for i := 0; i < 20480; i++ {
		pool.Put(make([]byte, 32*1024))
	}

	//pool.Purge()

	//time.Sleep(60 * time.Second)

	runtime.GC()

	var stat2 = &runtime.MemStats{}
	runtime.ReadMemStats(stat2)
	t.Log((stat2.HeapInuse-stat1.HeapInuse)/1024/1024, "MB,", pool.Size(), "slices")
}

func BenchmarkBytePool_Get(b *testing.B) {
	runtime.GOMAXPROCS(1)

	pool := NewBytePool(1024, 1)
	for i := 0; i < b.N; i++ {
		buf := pool.Get()
		_ = buf
		pool.Put(buf)
	}

	b.Log(pool.Size())
}
