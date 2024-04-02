// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package trackers_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/trackers"
	"github.com/iwind/TeaGo/logs"
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	{
		var tr = trackers.Begin("a")
		tr.End()
	}
	{
		var tr = trackers.Begin("a")
		time.Sleep(1 * time.Millisecond)
		tr.End()
	}
	{
		var tr = trackers.Begin("a")
		time.Sleep(2 * time.Millisecond)
		tr.End()
	}
	{
		var tr = trackers.Begin("a")
		time.Sleep(3 * time.Millisecond)
		tr.End()
	}
	{
		var tr = trackers.Begin("a")
		time.Sleep(4 * time.Millisecond)
		tr.End()
	}
	{
		var tr = trackers.Begin("a")
		time.Sleep(5 * time.Millisecond)
		tr.End()
	}
	{
		var tr = trackers.Begin("b")
		tr.End()
	}

	logs.PrintAsJSON(trackers.SharedManager.Labels(), t)
}

func TestTrackers_Add(t *testing.T) {
	var tr = trackers.Begin("a")
	time.Sleep(50 * time.Millisecond)
	tr.Add(-10 * time.Millisecond)
	tr.End()
	t.Log(trackers.SharedManager.Labels())
}
