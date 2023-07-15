// Copyright 2023 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package counters_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils/counters"
	"github.com/TeaOSLab/EdgeNode/internal/utils/testutils"
	"github.com/iwind/TeaGo/assert"
	timeutil "github.com/iwind/TeaGo/utils/time"
	"runtime"
	"testing"
	"time"
)

func TestItem_Increase(t *testing.T) {
	// run only under single testing
	if !testutils.IsSingleTesting() {
		return
	}

	var a = assert.NewAssertion(t)

	var item = counters.NewItem(20)
	for i := 0; i < 100; i++ {
		t.Log(item.Increase(), timeutil.Format("i:s"))
		time.Sleep(2 * time.Second)
	}

	item.Reset()
	a.IsTrue(item.Sum() == 0)
}

func TestItem_IsExpired(t *testing.T) {
	if !testutils.IsSingleTesting() {
		return
	}

	var currentTime = time.Now().Unix()

	var item = counters.NewItem(10)
	t.Log(item.IsExpired(currentTime))
	time.Sleep(10 * time.Second)
	t.Log(item.IsExpired(currentTime))
	time.Sleep(2 * time.Second)
	t.Log(item.IsExpired(currentTime))
}

func BenchmarkItem_Increase(b *testing.B) {
	runtime.GOMAXPROCS(1)

	var item = counters.NewItem(60)

	for i := 0; i < b.N; i++ {
		item.Increase()
	}
}
