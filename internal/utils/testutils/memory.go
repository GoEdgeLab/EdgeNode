// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package testutils

import (
	"fmt"
	timeutil "github.com/iwind/TeaGo/utils/time"
	"runtime"
	"testing"
	"time"
)

func StartMemoryStatsGC(t *testing.T) {
	var ticker = time.NewTicker(1 * time.Second)
	go func() {
		var stat = &runtime.MemStats{}
		var lastHeapInUse uint64

		for range ticker.C {
			runtime.ReadMemStats(stat)
			if stat.HeapInuse == lastHeapInUse {
				return
			}
			lastHeapInUse = stat.HeapInuse

			var before = time.Now()
			runtime.GC()
			var cost = time.Since(before).Seconds()

			t.Log(timeutil.Format("H:i:s"), "HeapInuse:", fmt.Sprintf("%.2fM", float64(stat.HeapInuse)/1024/1024), "NumGC:", stat.NumGC, "Cost:", fmt.Sprintf("%.4f", cost*1000), "ms")
		}
	}()
}

func StartMemoryStats(t *testing.T, callbacks ...func()) {
	var ticker = time.NewTicker(1 * time.Second)
	go func() {
		var stat = &runtime.MemStats{}
		var lastHeapInUse uint64

		for range ticker.C {
			runtime.ReadMemStats(stat)
			if stat.HeapInuse == lastHeapInUse {
				continue
			}
			lastHeapInUse = stat.HeapInuse

			t.Log(timeutil.Format("H:i:s"), "HeapInuse:", fmt.Sprintf("%.2fM", float64(stat.HeapInuse)/1024/1024), "NumGC:", stat.NumGC)

			if len(callbacks) > 0 {
				for _, callback := range callbacks {
					callback()
				}
			}
		}
	}()
}
