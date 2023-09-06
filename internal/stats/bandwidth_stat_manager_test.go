// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package stats_test

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeNode/internal/stats"
	"runtime"
	"testing"
	"time"
)

func TestBandwidthStatManager_Add(t *testing.T) {
	var manager = stats.NewBandwidthStatManager()
	manager.AddBandwidth(1, 0, 1, 10, 10)
	manager.AddBandwidth(1, 0, 1, 10, 10)
	manager.AddBandwidth(1, 0, 1, 10, 10)
	time.Sleep(1 * time.Second)
	manager.AddBandwidth(1, 0, 1, 85, 85)
	time.Sleep(1 * time.Second)
	manager.AddBandwidth(1, 0, 1, 25, 25)
	manager.AddBandwidth(1, 0, 1, 75, 75)
	manager.Inspect()
}

func TestBandwidthStatManager_Loop(t *testing.T) {
	var manager = stats.NewBandwidthStatManager()
	manager.AddBandwidth(1, 0, 1, 10, 10)
	manager.AddBandwidth(1, 0, 1, 10, 10)
	manager.AddBandwidth(1, 0, 1, 10, 10)
	err := manager.Loop()
	if err != nil {
		t.Fatal(err)
	}
}

func BenchmarkBandwidthStatManager_Add(b *testing.B) {
	var manager = stats.NewBandwidthStatManager()
	b.RunParallel(func(pb *testing.PB) {
		var i int
		for pb.Next() {
			i++
			manager.AddBandwidth(1, 0, int64(i%100), 10, 10)
		}
	})
}

func BenchmarkBandwidthStatManager_Slice(b *testing.B) {
	runtime.GOMAXPROCS(1)

	for i := 0; i < b.N; i++ {
		var pbStats = []*pb.ServerBandwidthStat{}
		for j := 0; j < 100; j++ {
			var stat = &stats.BandwidthStat{}
			pbStats = append(pbStats, &pb.ServerBandwidthStat{
				Id:                  0,
				UserId:              stat.UserId,
				ServerId:            stat.ServerId,
				Day:                 stat.Day,
				TimeAt:              stat.TimeAt,
				Bytes:               stat.MaxBytes / 2,
				TotalBytes:          stat.TotalBytes,
				CachedBytes:         stat.CachedBytes,
				AttackBytes:         stat.AttackBytes,
				CountRequests:       stat.CountRequests,
				CountCachedRequests: stat.CountCachedRequests,
				CountAttackRequests: stat.CountAttackRequests,
				NodeRegionId:        1,
			})
		}
		_ = pbStats
	}
}

func BenchmarkBandwidthStatManager_Slice2(b *testing.B) {
	runtime.GOMAXPROCS(1)

	for i := 0; i < b.N; i++ {
		var statsSlice = []*stats.BandwidthStat{}
		for j := 0; j < 100; j++ {
			var stat = &stats.BandwidthStat{}
			statsSlice = append(statsSlice, stat)
		}
		_ = statsSlice
	}
}

func BenchmarkBandwidthStatManager_Slice3(b *testing.B) {
	runtime.GOMAXPROCS(1)

	for i := 0; i < b.N; i++ {
		var statsSlice = make([]*stats.BandwidthStat, 2000)
		for j := 0; j < 100; j++ {
			var stat = &stats.BandwidthStat{}
			statsSlice[j] = stat
		}
	}
}
