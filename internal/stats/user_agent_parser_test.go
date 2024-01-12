// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package stats_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/stats"
	"github.com/TeaOSLab/EdgeNode/internal/utils/testutils"
	"github.com/iwind/TeaGo/assert"
	"github.com/iwind/TeaGo/rands"
	"github.com/iwind/TeaGo/types"
	"runtime"
	"runtime/debug"
	"testing"
	"time"
)

func TestUserAgentParser_Parse(t *testing.T) {
	var parser = stats.NewUserAgentParser()
	for i := 0; i < 4; i++ {
		t.Log(parser.Parse("Mozilla/5.0 (Windows NT 10.0; WOW64; rv:49.0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.96 Safari/537.36 Test/1"))
		t.Log(parser.Parse("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/96.0.4664.110 Safari/537.36"))
	}
}

func TestUserAgentParser_Parse_Unknown(t *testing.T) {
	var parser = stats.NewUserAgentParser()
	t.Log(parser.Parse("Mozilla/5.0 (Wind 10.0; WOW64; rv:49.0) Apple/537.36 (KHTML, like Gecko) Chr/88.0.4324.96 Sa/537.36 Test/1"))
	t.Log(parser.Parse(""))
}

func TestUserAgentParser_Memory(t *testing.T) {
	var stat1 = &runtime.MemStats{}
	runtime.ReadMemStats(stat1)

	var parser = stats.NewUserAgentParser()

	for i := 0; i < 1_000_000; i++ {
		parser.Parse("Mozilla/5.0 (Windows NT 10.0; WOW64; rv:49.0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.96 Safari/537.36 Test/" + types.String(rands.Int(0, 1_000_000)))
	}

	runtime.GC()
	debug.FreeOSMemory()

	var stat2 = &runtime.MemStats{}
	runtime.ReadMemStats(stat2)

	t.Log("max cache items:", parser.MaxCacheItems())
	t.Log("cache:", parser.Len(), "usage:", (stat2.HeapInuse-stat1.HeapInuse)>>20, "MB")
}

func TestNewUserAgentParser_GC(t *testing.T) {
	if !testutils.IsSingleTesting() {
		return
	}

	var parser = stats.NewUserAgentParser()

	for i := 0; i < 1_000_000; i++ {
		parser.Parse("Mozilla/5.0 (Windows NT 10.0; WOW64; rv:49.0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.96 Safari/537.36 Test/" + types.String(rands.Int(0, 100_000)))
	}

	time.Sleep(60 * time.Second) // wait to gc
	t.Log(parser.Len(), "cache items")
}

func TestNewUserAgentParser_Mobile(t *testing.T) {
	var a = assert.NewAssertion(t)
	var parser = stats.NewUserAgentParser()
	for _, userAgent := range []string{
		"Mozilla/5.0 (iPhone; CPU iPhone OS 12_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148",
		"Mozilla/5.0 (Linux; U; Android 2.2.1; en-us; Nexus One Build/FRG83) AppleWebKit/533.1 (KHTML, like Gecko) Version/4.0 Mobile Safari/533.1",
	} {
		a.IsTrue(parser.Parse(userAgent).IsMobile)
	}
}

func BenchmarkUserAgentParser_Parse_Many_LimitCPU(b *testing.B) {
	runtime.GOMAXPROCS(4)

	var parser = stats.NewUserAgentParser()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			parser.Parse("Mozilla/5.0 (Windows NT 10.0; WOW64; rv:49.0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.96 Safari/537.36 Test/" + types.String(rands.Int(0, 1_000_000)))
		}
	})
	b.Log(parser.Len())
}

func BenchmarkUserAgentParser_Parse_Many(b *testing.B) {
	var parser = stats.NewUserAgentParser()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			parser.Parse("Mozilla/5.0 (Windows NT 10.0; WOW64; rv:49.0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.96 Safari/537.36 Test/" + types.String(rands.Int(0, 1_000_000)))
		}
	})
	b.Log(parser.Len())
}

func BenchmarkUserAgentParser_Parse_Few_LimitCPU(b *testing.B) {
	runtime.GOMAXPROCS(4)

	var parser = stats.NewUserAgentParser()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			parser.Parse("Mozilla/5.0 (Windows NT 10.0; WOW64; rv:49.0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.96 Safari/537.36 Test/" + types.String(rands.Int(0, 100_000)))
		}
	})
	b.Log(parser.Len())
}

func BenchmarkUserAgentParser_Parse_Few(b *testing.B) {
	var parser = stats.NewUserAgentParser()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			parser.Parse("Mozilla/5.0 (Windows NT 10.0; WOW64; rv:49.0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.96 Safari/537.36 Test/" + types.String(rands.Int(0, 100_000)))
		}
	})
	b.Log(parser.Len())
}
