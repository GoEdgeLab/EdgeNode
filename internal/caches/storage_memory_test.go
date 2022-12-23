package caches

import (
	"bytes"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/iwind/TeaGo/logs"
	"github.com/iwind/TeaGo/rands"
	"runtime"
	"runtime/debug"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestMemoryStorage_OpenWriter(t *testing.T) {
	var storage = NewMemoryStorage(&serverconfigs.HTTPCachePolicy{}, nil)

	writer, err := storage.OpenWriter("abc", time.Now().Unix()+60, 200, -1, -1, -1, false)
	if err != nil {
		t.Fatal(err)
	}
	if err != nil {
		t.Fatal(err)
	}
	_, _ = writer.WriteHeader([]byte("Header"))
	_, _ = writer.Write([]byte("Hello"))
	_, _ = writer.Write([]byte(", World"))
	err = writer.Close()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(storage.valuesMap)

	{
		reader, err := storage.OpenReader("abc", false, false)
		if err != nil {
			if err == ErrNotFound {
				t.Log("not found: abc")
				return
			} else {
				t.Fatal(err)
			}
		}
		buf := make([]byte, 1024)
		t.Log("status:", reader.Status())
		err = reader.ReadHeader(buf, func(n int) (goNext bool, err error) {
			t.Log("header:", string(buf[:n]))
			return true, nil
		})
		if err != nil {
			t.Fatal(err)
		}
		err = reader.ReadBody(buf, func(n int) (goNext bool, err error) {
			t.Log("body:", string(buf[:n]))
			return true, nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	{
		_, err := storage.OpenReader("abc 2", false, false)
		if err != nil {
			if err == ErrNotFound {
				t.Log("not found: abc2")
			} else {
				t.Fatal(err)
			}
		}
	}

	writer, err = storage.OpenWriter("abc", time.Now().Unix()+60, 200, -1, -1, -1, false)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = writer.Write([]byte("Hello123"))
	{
		reader, err := storage.OpenReader("abc", false, false)
		if err != nil {
			if err == ErrNotFound {
				t.Log("not found: abc")
			} else {
				t.Fatal(err)
			}
		}
		buf := make([]byte, 1024)
		err = reader.ReadBody(buf, func(n int) (goNext bool, err error) {
			t.Log("abc:", string(buf[:n]))
			return true, nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestMemoryStorage_OpenReaderLock(t *testing.T) {
	storage := NewMemoryStorage(&serverconfigs.HTTPCachePolicy{}, nil)
	_ = storage.Init()

	var h = storage.hash("test")
	storage.valuesMap = map[uint64]*MemoryItem{
		h: {
			IsDone: true,
		},
	}
	_, _ = storage.OpenReader("test", false, false)
}

func TestMemoryStorage_Delete(t *testing.T) {
	var storage = NewMemoryStorage(&serverconfigs.HTTPCachePolicy{}, nil)
	{
		writer, err := storage.OpenWriter("abc", time.Now().Unix()+60, 200, -1, -1, -1, false)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = writer.Write([]byte("Hello"))
		err = writer.Close()
		if err != nil {
			t.Fatal(err)
		}
		t.Log(len(storage.valuesMap))
	}
	{
		writer, err := storage.OpenWriter("abc1", time.Now().Unix()+60, 200, -1, -1, -1, false)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = writer.Write([]byte("Hello"))
		err = writer.Close()
		if err != nil {
			t.Fatal(err)
		}
		t.Log(len(storage.valuesMap))
	}
	_ = storage.Delete("abc1")
	t.Log(len(storage.valuesMap))
}

func TestMemoryStorage_Stat(t *testing.T) {
	var storage = NewMemoryStorage(&serverconfigs.HTTPCachePolicy{}, nil)
	expiredAt := time.Now().Unix() + 60
	{
		writer, err := storage.OpenWriter("abc", expiredAt, 200, -1, -1, -1, false)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = writer.Write([]byte("Hello"))
		err = writer.Close()
		if err != nil {
			t.Fatal(err)
		}
		t.Log(len(storage.valuesMap))
		storage.AddToList(&Item{
			Key:       "abc",
			BodySize:  5,
			ExpiredAt: expiredAt,
		})
	}
	{
		writer, err := storage.OpenWriter("abc1", expiredAt, 200, -1, -1, -1, false)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = writer.Write([]byte("Hello"))
		err = writer.Close()
		if err != nil {
			t.Fatal(err)
		}
		t.Log(len(storage.valuesMap))
		storage.AddToList(&Item{
			Key:       "abc1",
			BodySize:  5,
			ExpiredAt: expiredAt,
		})
	}
	stat, err := storage.Stat()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("===stat===")
	logs.PrintAsJSON(stat, t)
}

func TestMemoryStorage_CleanAll(t *testing.T) {
	var storage = NewMemoryStorage(&serverconfigs.HTTPCachePolicy{}, nil)
	var expiredAt = time.Now().Unix() + 60
	{
		writer, err := storage.OpenWriter("abc", expiredAt, 200, -1, -1, -1, false)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = writer.Write([]byte("Hello"))
		err = writer.Close()
		if err != nil {
			t.Fatal(err)
		}
		storage.AddToList(&Item{
			Key:       "abc",
			BodySize:  5,
			ExpiredAt: expiredAt,
		})
	}
	{
		writer, err := storage.OpenWriter("abc1", expiredAt, 200, -1, -1, -1, false)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = writer.Write([]byte("Hello"))
		err = writer.Close()
		if err != nil {
			t.Fatal(err)
		}
		storage.AddToList(&Item{
			Key:       "abc1",
			BodySize:  5,
			ExpiredAt: expiredAt,
		})
	}
	err := storage.CleanAll()
	if err != nil {
		t.Fatal(err)
	}
	total, _ := storage.list.Count()
	t.Log(total, len(storage.valuesMap))
}

func TestMemoryStorage_Purge(t *testing.T) {
	storage := NewMemoryStorage(&serverconfigs.HTTPCachePolicy{}, nil)
	expiredAt := time.Now().Unix() + 60
	{
		writer, err := storage.OpenWriter("abc", expiredAt, 200, -1, -1, -1, false)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = writer.Write([]byte("Hello"))
		err = writer.Close()
		if err != nil {
			t.Fatal(err)
		}
		storage.AddToList(&Item{
			Key:       "abc",
			BodySize:  5,
			ExpiredAt: expiredAt,
		})
	}
	{
		writer, err := storage.OpenWriter("abc1", expiredAt, 200, -1, -1, -1, false)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = writer.Write([]byte("Hello"))
		err = writer.Close()
		if err != nil {
			t.Fatal(err)
		}
		storage.AddToList(&Item{
			Key:       "abc1",
			BodySize:  5,
			ExpiredAt: expiredAt,
		})
	}
	err := storage.Purge([]string{"abc", "abc1"}, "")
	if err != nil {
		t.Fatal(err)
	}
	total, _ := storage.list.Count()
	t.Log(total, len(storage.valuesMap))
}

func TestMemoryStorage_Expire(t *testing.T) {
	var storage = NewMemoryStorage(&serverconfigs.HTTPCachePolicy{
		MemoryAutoPurgeInterval: 5,
	}, nil)
	err := storage.Init()
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 1000; i++ {
		expiredAt := time.Now().Unix() + int64(rands.Int(0, 60))
		key := "abc" + strconv.Itoa(i)
		writer, err := storage.OpenWriter(key, expiredAt, 200, -1, -1, -1, false)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = writer.Write([]byte("Hello"))
		err = writer.Close()
		if err != nil {
			t.Fatal(err)
		}
		storage.AddToList(&Item{
			Key:       key,
			BodySize:  5,
			ExpiredAt: expiredAt,
		})
	}
	time.Sleep(70 * time.Second)
}

func TestMemoryStorage_Locker(t *testing.T) {
	var storage = NewMemoryStorage(&serverconfigs.HTTPCachePolicy{}, nil)
	err := storage.Init()
	if err != nil {
		t.Fatal(err)
	}
	storage.locker.Lock()
	err = storage.deleteWithoutLocker("a")
	storage.locker.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ok")
}

func TestMemoryStorage_Stop(t *testing.T) {
	var stat1 = &runtime.MemStats{}
	runtime.ReadMemStats(stat1)

	var m = map[uint64]*MemoryItem{}
	for i := 0; i < 1_000_000; i++ {
		m[uint64(i)] = &MemoryItem{
			HeaderValue: []byte("Hello, World"),
			BodyValue:   bytes.Repeat([]byte("Hello"), 1024),
		}
	}

	m = map[uint64]*MemoryItem{}

	var before = time.Now()
	//runtime.GC()
	debug.FreeOSMemory()
	/**go func() {
		time.Sleep(10 * time.Second)
		runtime.GC()
	}()**/
	t.Log(time.Since(before).Seconds()*1000, "ms")

	var stat2 = &runtime.MemStats{}
	runtime.ReadMemStats(stat2)

	if stat2.HeapInuse > stat1.HeapInuse {
		t.Log(stat2.HeapInuse, stat1.HeapInuse, (stat2.HeapInuse-stat1.HeapInuse)/1024/1024, "MB")
	} else {
		t.Log("0 MB")
	}

	t.Log(len(m))
}

func BenchmarkValuesMap(b *testing.B) {
	var m = map[uint64]*MemoryItem{}
	var count = 1_000_000
	for i := 0; i < count; i++ {
		m[uint64(i)] = &MemoryItem{
			ExpiresAt: time.Now().Unix(),
		}
	}
	b.Log(len(m))

	var locker = sync.Mutex{}
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			locker.Lock()
			_, ok := m[uint64(rands.Int(0, 1_000_000))]
			_ = ok
			locker.Unlock()

			locker.Lock()
			delete(m, uint64(rands.Int(2, 1000000)))
			locker.Unlock()
		}
	})
}
