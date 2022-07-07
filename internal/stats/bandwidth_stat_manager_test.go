// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package stats_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/stats"
	"testing"
	"time"
)

func TestBandwidthStatManager_Add(t *testing.T) {
	var manager = stats.NewBandwidthStatManager()
	manager.Add(1, 1, 10)
	manager.Add(1, 1, 10)
	manager.Add(1, 1, 10)
	time.Sleep(1 * time.Second)
	manager.Add(1, 1, 15)
	time.Sleep(1 * time.Second)
	manager.Add(1, 1, 25)
	manager.Add(1, 1, 75)
	manager.Inspect()
}

func TestBandwidthStatManager_Loop(t *testing.T) {
	var manager = stats.NewBandwidthStatManager()
	manager.Add(1, 1, 10)
	manager.Add(1, 1, 10)
	manager.Add(1, 1, 10)
	err := manager.Loop()
	if err != nil {
		t.Fatal(err)
	}
}
