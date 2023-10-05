// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package caches_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/caches"
	"github.com/TeaOSLab/EdgeNode/internal/zero"
	"github.com/iwind/TeaGo/Tea"
	"github.com/iwind/TeaGo/rands"
	"github.com/iwind/TeaGo/types"
	stringutil "github.com/iwind/TeaGo/utils/string"
	"math/big"
	"runtime"
	"strconv"
	"testing"
	"time"
)

func TestFileListHashMap_Memory(t *testing.T) {
	var stat1 = &runtime.MemStats{}
	runtime.ReadMemStats(stat1)

	var m = caches.NewFileListHashMap()
	m.SetIsAvailable(true)

	for i := 0; i < 1_000_000; i++ {
		m.Add(stringutil.Md5(types.String(i)))
	}

	t.Log("added:", m.Len(), "hashes")

	var stat2 = &runtime.MemStats{}
	runtime.ReadMemStats(stat2)

	t.Log("ready", (stat2.Alloc-stat1.Alloc)/1024/1024, "M")
	t.Log("remains:", m.Len(), "hashes")
}

func TestFileListHashMap_Memory2(t *testing.T) {
	var stat1 = &runtime.MemStats{}
	runtime.ReadMemStats(stat1)

	var m = map[uint64]zero.Zero{}

	for i := 0; i < 1_000_000; i++ {
		m[uint64(i)] = zero.New()
	}

	var stat2 = &runtime.MemStats{}
	runtime.ReadMemStats(stat2)

	t.Log("ready", (stat2.Alloc-stat1.Alloc)/1024/1024, "M")
}

func TestFileListHashMap_BigInt(t *testing.T) {
	var bigInt = big.NewInt(0)

	for _, s := range []string{"1", "2", "3", "123", "123456"} {
		var hash = stringutil.Md5(s)

		var bigInt1 = big.NewInt(0)
		bigInt1.SetString(hash, 16)

		bigInt.SetString(hash, 16)

		t.Log(s, "=>", bigInt1.Uint64(), "hash:", hash, "format:", strconv.FormatUint(bigInt1.Uint64(), 16), strconv.FormatUint(bigInt.Uint64(), 16))

		if strconv.FormatUint(bigInt1.Uint64(), 16) != strconv.FormatUint(bigInt.Uint64(), 16) {
			t.Fatal("not equal")
		}
	}

	for i := 0; i < 1_000_000; i++ {
		var hash = stringutil.Md5(types.String(i))

		var bigInt1 = big.NewInt(0)
		bigInt1.SetString(hash, 16)

		bigInt.SetString(hash, 16)

		if bigInt1.Uint64() != bigInt.Uint64() {
			t.Fatal(i, "not equal")
		}
	}
}

func TestFileListHashMap_Load(t *testing.T) {
	var list = caches.NewFileList(Tea.Root + "/data/cache-index/p1").(*caches.FileList)

	defer func() {
		_ = list.Close()
	}()

	err := list.Init()
	if err != nil {
		t.Fatal(err)
	}

	var m = caches.NewFileListHashMap()
	var before = time.Now()
	var db = list.GetDB("abc")
	err = m.Load(db)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(time.Since(before).Seconds()*1000, "ms")
	t.Log("count:", m.Len())
	m.Add("abc")

	for _, hash := range []string{"33347bb4441265405347816cad36a0f8", "a", "abc", "123"} {
		t.Log(hash, "=>", m.Exist(hash))
	}
}

func Benchmark_BigInt(b *testing.B) {
	var hash = stringutil.Md5("123456")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var bigInt = big.NewInt(0)
		bigInt.SetString(hash, 16)
		_ = bigInt.Uint64()
	}
}

func BenchmarkFileListHashMap_Exist(b *testing.B) {
	var m = caches.NewFileListHashMap()
	m.SetIsAvailable(true)
	m.SetIsReady(true)

	for i := 0; i < 1_000_000; i++ {
		m.Add(types.String(i))
	}

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			m.Add(types.String(rands.Int64()))
			_ = m.Exist(types.String(rands.Int64()))
		}
	})
}
