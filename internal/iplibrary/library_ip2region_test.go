package iplibrary

import (
	"github.com/iwind/TeaGo/Tea"
	_ "github.com/iwind/TeaGo/bootstrap"
	"github.com/iwind/TeaGo/rands"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestIP2RegionLibrary_Lookup_MemoryUsage(t *testing.T) {
	var mem = &runtime.MemStats{}
	runtime.ReadMemStats(mem)

	library := &IP2RegionLibrary{}
	err := library.Load(Tea.Root + "/resources/ipdata/ip2region/ip2region.db")
	if err != nil {
		t.Fatal(err)
	}

	var mem2 = &runtime.MemStats{}
	runtime.ReadMemStats(mem2)
	t.Log((mem2.HeapInuse-mem.HeapInuse)/1024/1024, "MB")
}

func TestIP2RegionLibrary_Lookup_Single(t *testing.T) {
	library := &IP2RegionLibrary{}
	err := library.Load(Tea.Root + "/resources/ipdata/ip2region/ip2region.db")
	if err != nil {
		t.Fatal(err)
	}

	for _, ip := range []string{"8.8.9.9"} {
		result, err := library.Lookup(ip)
		if err != nil {
			t.Fatal(err)
		}
		t.Log("IP:", ip, "result:", result)
	}
}

func TestIP2RegionLibrary_Lookup(t *testing.T) {
	library := &IP2RegionLibrary{}
	err := library.Load(Tea.Root + "/resources/ipdata/ip2region/ip2region.db")
	if err != nil {
		t.Fatal(err)
	}

	for _, ip := range []string{"", "a", "1.1.1", "192.168.1.100", "114.240.223.47", "8.8.9.9", "::1"} {
		result, err := library.Lookup(ip)
		if err != nil {
			t.Fatal(err)
		}
		t.Log("IP:", ip, "result:", result)
	}
}

func TestIP2RegionLibrary_Lookup_Concurrent(t *testing.T) {
	library := &IP2RegionLibrary{}
	err := library.Load(Tea.Root + "/resources/ipdata/ip2region/ip2region.db")
	if err != nil {
		t.Fatal(err)
	}

	var count = 4000
	var wg = sync.WaitGroup{}
	wg.Add(count)
	for i := 0; i < count; i++ {
		go func() {
			defer wg.Done()

			for i := 0; i < 100; i++ {
				_, _ = library.Lookup(strconv.Itoa(rands.Int(0, 254)) + "." + strconv.Itoa(rands.Int(0, 254)) + "." + strconv.Itoa(rands.Int(0, 254)) + "." + strconv.Itoa(rands.Int(0, 254)))
			}
		}()
	}

	wg.Done()
	t.Log("ok")
}

func TestIP2RegionLibrary_Memory(t *testing.T) {
	library := &IP2RegionLibrary{}
	err := library.Load(Tea.Root + "/resources/ipdata/ip2region/ip2region.db")
	if err != nil {
		t.Fatal(err)
	}

	before := time.Now()

	for i := 0; i < 1_000_000; i++ {
		_, _ = library.Lookup(strconv.Itoa(rands.Int(0, 254)) + "." + strconv.Itoa(rands.Int(0, 254)) + "." + strconv.Itoa(rands.Int(0, 254)) + "." + strconv.Itoa(rands.Int(0, 254)))
	}

	t.Log("cost:", time.Since(before).Seconds()*1000, "ms")
}

func BenchmarkIP2RegionLibrary_Lookup(b *testing.B) {
	runtime.GOMAXPROCS(1)

	var library = &IP2RegionLibrary{}
	err := library.Load(Tea.Root + "/resources/ipdata/ip2region/ip2region.db")
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = library.Lookup("8.8.8.8")
	}
}
