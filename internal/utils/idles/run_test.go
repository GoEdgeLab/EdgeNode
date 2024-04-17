// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package idles_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils/idles"
	"github.com/TeaOSLab/EdgeNode/internal/utils/testutils"
	"github.com/iwind/TeaGo/logs"
	timeutil "github.com/iwind/TeaGo/utils/time"
	"testing"
	"time"
)

func TestCheckHourlyLoad(t *testing.T) {
	for i := 0; i < 10; i++ {
		idles.CheckHourlyLoad(1)
		idles.CheckHourlyLoad(2)
		idles.CheckHourlyLoad(3)
	}

	t.Log(idles.TestMinLoadHour())
	logs.PrintAsJSON(idles.TestHourlyLoadMap(), t)
}

func TestRun(t *testing.T) {
	//idles.CheckHourlyLoad(time.Now().Hour())
	idles.Run(func() {
		t.Log("run once")
	})
}

func TestRunTicker(t *testing.T) {
	if !testutils.IsSingleTesting() {
		return
	}

	var ticker = time.NewTicker(10 * time.Second)
	idles.RunTicker(ticker, func() {
		t.Log(timeutil.Format("H:i:s"), "run once")
	})
}
