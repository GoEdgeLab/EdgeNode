// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package nodes

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/nodeconfigs"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/TeaOSLab/EdgeNode/internal/monitor"
	"github.com/iwind/TeaGo/maps"
	"sync/atomic"
	"time"
)

// 发送监控流量
func init() {
	if !teaconst.IsMain {
		return
	}

	events.On(events.EventStart, func() {
		var ticker = time.NewTicker(1 * time.Minute)
		goman.New(func() {
			for range ticker.C {
				// 加入到数据队列中
				var inBytes = atomic.LoadUint64(&teaconst.InTrafficBytes)
				atomic.StoreUint64(&teaconst.InTrafficBytes, 0) // 重置数据
				if inBytes > 0 {
					monitor.SharedValueQueue.Add(nodeconfigs.NodeValueItemTrafficIn, maps.Map{
						"total": inBytes,
					})
				}

				var outBytes = atomic.LoadUint64(&teaconst.OutTrafficBytes)
				atomic.StoreUint64(&teaconst.OutTrafficBytes, 0) // 重置数据
				if outBytes > 0 {
					monitor.SharedValueQueue.Add(nodeconfigs.NodeValueItemTrafficOut, maps.Map{
						"total": outBytes,
					})
				}
			}
		})
	})
}
