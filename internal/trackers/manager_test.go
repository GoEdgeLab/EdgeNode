// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package trackers

import (
	"github.com/iwind/TeaGo/logs"
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	{
		var tr = Begin("a")
		tr.End()
	}
	{
		var tr = Begin("a")
		time.Sleep(1 * time.Millisecond)
		tr.End()
	}
	{
		var tr = Begin("a")
		time.Sleep(2 * time.Millisecond)
		tr.End()
	}
	{
		var tr = Begin("a")
		time.Sleep(3 * time.Millisecond)
		tr.End()
	}
	{
		var tr = Begin("a")
		time.Sleep(4 * time.Millisecond)
		tr.End()
	}
	{
		var tr = Begin("a")
		time.Sleep(5 * time.Millisecond)
		tr.End()
	}
	{
		var tr = Begin("b")
		tr.End()
	}

	logs.PrintAsJSON(SharedManager.Labels(), t)
}
