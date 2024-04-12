// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package stats_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/stats"
	"github.com/TeaOSLab/EdgeNode/internal/utils/testutils"
	"github.com/iwind/TeaGo/rands"
	"runtime"
	"testing"
	"time"
)

func TestDAUManager_AddIP(t *testing.T) {
	var manager = stats.NewDAUManager()
	err := manager.Init()
	if err != nil {
		t.Fatal(err)
	}

	manager.AddIP(1, "127.0.0.1")
	manager.AddIP(1, "127.0.0.2")
	manager.AddIP(1, "127.0.0.3")
	manager.AddIP(1, "127.0.0.4")
	manager.AddIP(1, "127.0.0.2")
	manager.AddIP(1, "127.0.0.3")

	time.Sleep(1 * time.Second)

	err = manager.Close()
	if err != nil {
		t.Fatal(err)
	}

	t.Log("======")
	manager.TestInspect(t)
}

func TestDAUManager_AddIP_Many(t *testing.T) {
	var manager = stats.NewDAUManager()
	err := manager.Init()
	if err != nil {
		t.Fatal(err)
	}

	var before = time.Now()
	defer func() {
		t.Log("cost:", time.Since(before).Seconds()*1000, "ms")
	}()

	var count = 1

	if testutils.IsSingleTesting() {
		count = 10_000
	}

	for i := 0; i < count; i++ {
		manager.AddIP(int64(rands.Int(1, 10)), testutils.RandIP())
	}
}

func TestDAUManager_CleanStats(t *testing.T) {
	var manager = stats.NewDAUManager()
	err := manager.Init()
	if err != nil {
		t.Fatal(err)
	}

	var before = time.Now()
	defer func() {
		t.Log("cost:", time.Since(before).Seconds()*1000, "ms")
	}()

	defer func() {
		_ = manager.Flush()
	}()

	err = manager.CleanStats()
	if err != nil {
		t.Fatal(err)
	}
}

func TestDAUManager_TestInspect(t *testing.T) {
	var manager = stats.NewDAUManager()
	err := manager.Init()
	if err != nil {
		t.Fatal(err)
	}

	manager.TestInspect(t)
}

func TestDAUManager_Truncate(t *testing.T) {
	var manager = stats.NewDAUManager()
	err := manager.Init()
	if err != nil {
		t.Fatal(err)
	}

	err = manager.Truncate()
	if err != nil {
		t.Fatal(err)
	}

	err = manager.Flush()
	if err != nil {
		t.Fatal(err)
	}
}

func BenchmarkDAUManager_AddIP_Cache(b *testing.B) {
	runtime.GOMAXPROCS(1)

	var cachedIPs []stats.IPInfo
	var maxIPs = 128
	b.Log("maxIPs:", maxIPs)
	for i := 0; i < maxIPs; i++ {
		cachedIPs = append(cachedIPs, stats.IPInfo{
			IP:       testutils.RandIP(),
			ServerId: 1,
		})
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var ip = "1.2.3.4"
		for _, cacheIP := range cachedIPs {
			if cacheIP.IP == ip && cacheIP.ServerId == 1 {
				break
			}
		}
	}
}
