package bytepool_test

import (
	"bytes"
	"github.com/TeaOSLab/EdgeNode/internal/utils/bytepool"
	"runtime"
	"sync"
	"testing"
)

func TestBytePool_Memory(t *testing.T) {
	var stat1 = &runtime.MemStats{}
	runtime.ReadMemStats(stat1)

	var pool = bytepool.NewPool(32 * 1024)
	for i := 0; i < 20480; i++ {
		pool.Put(&bytepool.Buf{
			Bytes: make([]byte, 32*1024),
		})
	}

	//pool.Purge()

	//time.Sleep(60 * time.Second)

	runtime.GC()

	var stat2 = &runtime.MemStats{}
	runtime.ReadMemStats(stat2)
	t.Log((stat2.HeapInuse-stat1.HeapInuse)/1024/1024, "MB,")
}

func BenchmarkBytePool_Get(b *testing.B) {
	runtime.GOMAXPROCS(1)

	var pool = bytepool.NewPool(1)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var buf = pool.Get()
		_ = buf
		pool.Put(buf)
	}
}

func BenchmarkBytePool_Get_Parallel(b *testing.B) {
	runtime.GOMAXPROCS(1)

	var pool = bytepool.NewPool(1024)
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var buf = pool.Get()
			pool.Put(buf)
		}
	})
}

func BenchmarkBytePool_Get_Sync(b *testing.B) {
	runtime.GOMAXPROCS(1)

	var pool = &sync.Pool{
		New: func() any {
			return make([]byte, 1024)
		},
	}

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var buf = pool.Get()
			pool.Put(buf)
		}
	})
}

func BenchmarkBytePool_Get_Sync2(b *testing.B) {
	runtime.GOMAXPROCS(1)

	var pool = &sync.Pool{
		New: func() any {
			return &bytepool.Buf{
				Bytes: make([]byte, 1024),
			}
		},
	}

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var buf = pool.Get()
			pool.Put(buf)
		}
	})
}

func BenchmarkBytePool_Copy_Bytes_4(b *testing.B) {
	const size = 4 << 10

	var data = bytes.Repeat([]byte{'A'}, size)

	var pool = &sync.Pool{
		New: func() any {
			return make([]byte, size)
		},
	}
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var buf = pool.Get().([]byte)
			copy(buf, data)
			pool.Put(buf)
		}
	})
}

func BenchmarkBytePool_Copy_Wrapper_4(b *testing.B) {
	const size = 4 << 10

	var data = bytes.Repeat([]byte{'A'}, size)

	var pool = &sync.Pool{
		New: func() any {
			return &bytepool.Buf{
				Bytes: make([]byte, size),
			}
		},
	}
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var buf = pool.Get().(*bytepool.Buf)
			copy(buf.Bytes, data)
			pool.Put(buf)
		}
	})
}

func BenchmarkBytePool_Copy_Bytes_16(b *testing.B) {
	const size = 16 << 10

	var data = bytes.Repeat([]byte{'A'}, size)

	var pool = &sync.Pool{
		New: func() any {
			return make([]byte, size)
		},
	}
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var buf = pool.Get().([]byte)
			copy(buf, data)
			pool.Put(buf)
		}
	})
}

func BenchmarkBytePool_Copy_Wrapper_16(b *testing.B) {
	const size = 16 << 10

	var data = bytes.Repeat([]byte{'A'}, size)

	var pool = &sync.Pool{
		New: func() any {
			return &bytepool.Buf{
				Bytes: make([]byte, size),
			}
		},
	}
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var buf = pool.Get().(*bytepool.Buf)
			copy(buf.Bytes, data)
			pool.Put(buf)
		}
	})
}

func BenchmarkBytePool_Copy_Wrapper_Buf_16(b *testing.B) {
	const size = 16 << 10

	var data = bytes.Repeat([]byte{'A'}, size)

	var pool = &sync.Pool{
		New: func() any {
			return &bytepool.Buf{
				Bytes: make([]byte, size),
			}
		},
	}
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var bytesPtr = pool.Get().(*bytepool.Buf)
			var buf = bytesPtr.Bytes
			copy(buf, data)
			pool.Put(bytesPtr)
		}
	})
}

func BenchmarkBytePool_Copy_Wrapper_BytePool_16(b *testing.B) {
	const size = 16 << 10

	var data = bytes.Repeat([]byte{'A'}, size)

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var bytesPtr = bytepool.Pool16k.Get()
			copy(bytesPtr.Bytes, data)
			bytepool.Pool16k.Put(bytesPtr)
		}
	})
}

func BenchmarkBytePool_Copy_Bytes_32(b *testing.B) {
	const size = 32 << 10

	var data = bytes.Repeat([]byte{'A'}, size)

	var pool = &sync.Pool{
		New: func() any {
			return make([]byte, size)
		},
	}
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var buf = pool.Get().([]byte)
			copy(buf, data)
			pool.Put(buf)
		}
	})
}

func BenchmarkBytePool_Copy_Wrapper_32(b *testing.B) {
	const size = 32 << 10

	var data = bytes.Repeat([]byte{'A'}, size)

	var pool = &sync.Pool{
		New: func() any {
			return &bytepool.Buf{
				Bytes: make([]byte, size),
			}
		},
	}
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var buf = pool.Get().(*bytepool.Buf)
			copy(buf.Bytes, data)
			pool.Put(buf)
		}
	})
}
