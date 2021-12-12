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
	events.On(events.EventStart, func() {
		ticker := time.NewTicker(1 * time.Minute)
		goman.New(func() {
			for range ticker.C {
				// 加入到数据队列中
				if teaconst.InTrafficBytes > 0 {
					monitor.SharedValueQueue.Add(nodeconfigs.NodeValueItemTrafficIn, maps.Map{
						"total": teaconst.InTrafficBytes,
					})
				}
				if teaconst.OutTrafficBytes > 0 {
					monitor.SharedValueQueue.Add(nodeconfigs.NodeValueItemTrafficOut, maps.Map{
						"total": teaconst.OutTrafficBytes,
					})
				}

				// 重置数据
				atomic.StoreUint64(&teaconst.InTrafficBytes, 0)
				atomic.StoreUint64(&teaconst.OutTrafficBytes, 0)
			}
		})
	})
}
