// Copyright 2023 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package counters_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils/counters"
	"github.com/TeaOSLab/EdgeNode/internal/utils/testutils"
	"github.com/iwind/TeaGo/assert"
	"github.com/iwind/TeaGo/types"
	timeutil "github.com/iwind/TeaGo/utils/time"
	"runtime"
	"testing"
	"time"
)

func TestItem_Increase(t *testing.T) {
	if !testutils.IsSingleTesting() {
		return
	}

	var item = counters.NewItem[uint32](10)
	t.Log(item.Increase(), item.Sum())
	time.Sleep(1 * time.Second)
	t.Log(item.Increase(), item.Sum())
	time.Sleep(2 * time.Second)
	t.Log(item.Increase(), item.Sum())
	time.Sleep(5 * time.Second)
	t.Log(item.Increase(), item.Sum())
	time.Sleep(6 * time.Second)
	t.Log(item.Increase(), item.Sum())
	time.Sleep(5 * time.Second)
	t.Log(item.Increase(), item.Sum())
	time.Sleep(11 * time.Second)
	t.Log(item.Increase(), item.Sum())
}

func TestItem_Increase2(t *testing.T) {
	// run only under single testing
	if !testutils.IsSingleTesting() {
		return
	}

	var a = assert.NewAssertion(t)

	var item = counters.NewItem[uint32](23)
	for i := 0; i < 100; i++ {
		t.Log("round "+types.String(i)+":", item.Increase(), item.Sum(), timeutil.Format("H:i:s"))
		time.Sleep(2 * time.Second)
	}

	item.Reset()
	a.IsTrue(item.Sum() == 0)
}

func TestItem_IsExpired(t *testing.T) {
	if !testutils.IsSingleTesting() {
		return
	}

	var item = counters.NewItem[uint32](10)
	t.Log(item.IsExpired(time.Now().Unix()))
	time.Sleep(10 * time.Second)
	t.Log(item.IsExpired(time.Now().Unix()))
	time.Sleep(2 * time.Second)
	t.Log(item.IsExpired(time.Now().Unix()))
	time.Sleep(2 * time.Second)
	t.Log(item.IsExpired(time.Now().Unix()))
}

func BenchmarkItem_Increase(b *testing.B) {
	runtime.GOMAXPROCS(1)

	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var item = counters.NewItem[uint32](60)
			item.Increase()
			item.Sum()
		}
	})
}
