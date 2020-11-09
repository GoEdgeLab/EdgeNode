package iplibrary

import (
	"github.com/iwind/TeaGo/Tea"
	_ "github.com/iwind/TeaGo/bootstrap"
	"github.com/iwind/TeaGo/logs"
	"github.com/iwind/TeaGo/rands"
	"runtime"
	"strconv"
	"testing"
	"time"
)

func TestIP2RegionLibrary_Lookup(t *testing.T) {
	library := &IP2RegionLibrary{}
	err := library.Load(Tea.Root + "/resources/ipdata/ip2region/ip2region.db")
	if err != nil {
		t.Fatal(err)
	}
	result, err := library.Lookup("114.240.223.47")
	if err != nil {
		t.Fatal(err)
	}
	logs.PrintAsJSON(result, t)
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

	library := &IP2RegionLibrary{}
	err := library.Load(Tea.Root + "/resources/ipdata/ip2region/ip2region.db")
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		_, _ = library.Lookup(strconv.Itoa(rands.Int(0, 254)) + "." + strconv.Itoa(rands.Int(0, 254)) + "." + strconv.Itoa(rands.Int(0, 254)) + "." + strconv.Itoa(rands.Int(0, 254)))
	}
}
