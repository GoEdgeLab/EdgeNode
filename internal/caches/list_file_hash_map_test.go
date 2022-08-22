// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package caches_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/caches"
	"github.com/TeaOSLab/EdgeNode/internal/zero"
	"github.com/iwind/TeaGo/Tea"
	"github.com/iwind/TeaGo/types"
	stringutil "github.com/iwind/TeaGo/utils/string"
	"math/big"
	"runtime"
	"strconv"
	"testing"
)

func TestFileListHashMap_Memory(t *testing.T) {
	var stat1 = &runtime.MemStats{}
	runtime.ReadMemStats(stat1)

	var m = caches.NewFileListHashMap()

	for i := 0; i < 1_000_000; i++ {
		m.Add(stringutil.Md5(types.String(i)))
	}

	var stat2 = &runtime.MemStats{}
	runtime.ReadMemStats(stat2)

	t.Log("ready", (stat2.Alloc-stat1.Alloc)/1024/1024, "M")
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
	for _, s := range []string{"1", "2", "3", "123", "123456"} {
		var hash = stringutil.Md5(s)

		var bigInt = big.NewInt(0)
		bigInt.SetString(hash, 16)
		t.Log(s, "=>", bigInt.Uint64(), "hash:", hash, "format:", strconv.FormatUint(bigInt.Uint64(), 16))
	}
}

func TestFileListHashMap_Load(t *testing.T) {
	var list = caches.NewFileList(Tea.Root + "/data/cache-index/p1").(*caches.FileList)
	err := list.Init()
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		_ = list.Close()
	}()

	var m = caches.NewFileListHashMap()
	err = m.Load(list.GetDB("abc"))
	if err != nil {
		t.Fatal(err)
	}
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
