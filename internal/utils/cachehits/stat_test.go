// Copyright 2023 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package cachehits_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils/cachehits"
	"github.com/TeaOSLab/EdgeNode/internal/utils/testutils"
	"github.com/iwind/TeaGo/assert"
	"github.com/iwind/TeaGo/rands"
	"github.com/iwind/TeaGo/types"
	"runtime"
	"strconv"
	"testing"
	"time"
)

func TestNewStat(t *testing.T) {
	var a = assert.NewAssertion(t)

	{
		var stat = cachehits.NewStat(20)
		for i := 0; i < 1000; i++ {
			stat.IncreaseCached("a")
		}

		a.IsTrue(stat.IsGood("a"))
	}

	{
		var stat = cachehits.NewStat(5)
		for i := 0; i < 10000; i++ {
			stat.IncreaseCached("a")
		}
		for i := 0; i < 500; i++ {
			stat.IncreaseHit("a")
		}

		stat.IncreaseHit("b") // empty

		a.IsTrue(stat.IsGood("a"))
		a.IsTrue(stat.IsGood("b"))
	}

	{
		var stat = cachehits.NewStat(10)
		for i := 0; i < 10000; i++ {
			stat.IncreaseCached("a")
		}
		for i := 0; i < 1000; i++ {
			stat.IncreaseHit("a")
		}

		stat.IncreaseHit("b") // empty

		a.IsTrue(stat.IsGood("a"))
		a.IsTrue(stat.IsGood("b"))
	}

	{
		var stat = cachehits.NewStat(5)
		for i := 0; i < 10001; i++ {
			stat.IncreaseCached("a")
		}
		for i := 0; i < 499; i++ {
			stat.IncreaseHit("a")
		}

		a.IsFalse(stat.IsGood("a"))
	}
}

func TestNewStat_Memory(t *testing.T) {
	if !testutils.IsSingleTesting() {
		return
	}

	var stat = cachehits.NewStat(20)
	for i := 0; i < 10_000_000; i++ {
		stat.IncreaseCached("a" + types.String(i))
	}

	time.Sleep(60 * time.Second)

	t.Log(stat.Len())
}

func BenchmarkStat(b *testing.B) {
	runtime.GOMAXPROCS(4)

	var stat = cachehits.NewStat(5)
	for i := 0; i < 1_000_000; i++ {
		stat.IncreaseCached("a" + types.String(i))
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var key = strconv.Itoa(rands.Int(0, 100_000))
			stat.IncreaseCached(key)
			if rands.Int(0, 3) == 0 {
				stat.IncreaseHit(key)
			}
			_ = stat.IsGood(key)
		}
	})
}
