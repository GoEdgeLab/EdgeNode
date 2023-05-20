// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package stats

import (
	"github.com/iwind/TeaGo/rands"
	"github.com/iwind/TeaGo/types"
	"runtime"
	"runtime/debug"
	"testing"
)

func TestUserAgentParser_Parse(t *testing.T) {
	var parser = NewUserAgentParser()
	for i := 0; i < 4; i++ {
		t.Log(parser.Parse("Mozilla/5.0 (Windows NT 10.0; WOW64; rv:49.0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.96 Safari/537.36 Test/1"))
		t.Log(parser.Parse("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/96.0.4664.110 Safari/537.36"))
	}
}

func TestUserAgentParser_Parse_Unknown(t *testing.T) {
	var parser = NewUserAgentParser()
	t.Log(parser.Parse("Mozilla/5.0 (Wind 10.0; WOW64; rv:49.0) Apple/537.36 (KHTML, like Gecko) Chr/88.0.4324.96 Sa/537.36 Test/1"))
	t.Log(parser.Parse(""))
}

func TestUserAgentParser_Memory(t *testing.T) {
	var stat1 = &runtime.MemStats{}
	runtime.ReadMemStats(stat1)

	var parser = NewUserAgentParser()

	for i := 0; i < 1_000_000; i++ {
		parser.Parse("Mozilla/5.0 (Windows NT 10.0; WOW64; rv:49.0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.96 Safari/537.36 Test/" + types.String(rands.Int(0, 100_000)))
	}

	runtime.GC()
	debug.FreeOSMemory()

	var stat2 = &runtime.MemStats{}
	runtime.ReadMemStats(stat2)

	t.Log("max cache items:", parser.maxCacheItems)
	t.Log("cache1:", len(parser.cacheMap1), "cache2:", len(parser.cacheMap2), "cache3:", (stat2.HeapInuse-stat1.HeapInuse)/1024/1024, "MB")
}

func BenchmarkUserAgentParser_Parse(b *testing.B) {
	var parser = NewUserAgentParser()
	for i := 0; i < b.N; i++ {
		parser.Parse("Mozilla/5.0 (Windows NT 10.0; WOW64; rv:49.0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.96 Safari/537.36 Test/" + types.String(rands.Int(0, 1_000_000)))
	}
	b.Log(len(parser.cacheMap1), len(parser.cacheMap2))
}

func BenchmarkUserAgentParser_Parse2(b *testing.B) {
	var parser = NewUserAgentParser()
	for i := 0; i < b.N; i++ {
		parser.Parse("Mozilla/5.0 (Windows NT 10.0; WOW64; rv:49.0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.96 Safari/537.36 Test/" + types.String(rands.Int(0, 100_000)))
	}
	b.Log(len(parser.cacheMap1), len(parser.cacheMap2))
}

func BenchmarkUserAgentParser_Parse3(b *testing.B) {
	var parser = NewUserAgentParser()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			parser.Parse("Mozilla/5.0 (Windows NT 10.0; WOW64; rv:49.0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.96 Safari/537.36 Test/" + types.String(rands.Int(0, 100_000)))
		}
	})
	b.Log(len(parser.cacheMap1), len(parser.cacheMap2))
}
