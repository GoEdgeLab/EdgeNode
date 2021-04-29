// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package nodes

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/nodeconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/monitor"
	"github.com/iwind/TeaGo/maps"
	"net"
	"sync/atomic"
	"time"
)

// 流量统计
var inTrafficBytes = uint64(0)
var outTrafficBytes = uint64(0)

// 发送监控流量
func init() {
	events.On(events.EventStart, func() {
		ticker := time.NewTicker(1 * time.Minute)
		go func() {
			for range ticker.C {
				// 加入到数据队列中
				if inTrafficBytes > 0 {
					monitor.SharedValueQueue.Add(nodeconfigs.NodeValueItemTrafficIn, maps.Map{
						"total": inTrafficBytes,
					})
				}
				if outTrafficBytes > 0 {
					monitor.SharedValueQueue.Add(nodeconfigs.NodeValueItemTrafficOut, maps.Map{
						"total": outTrafficBytes,
					})
				}

				// 重置数据
				atomic.StoreUint64(&inTrafficBytes, 0)
				atomic.StoreUint64(&outTrafficBytes, 0)
			}
		}()
	})
}

// TrafficConn 用于统计流量的连接
type TrafficConn struct {
	rawConn net.Conn
}

func NewTrafficConn(conn net.Conn) net.Conn {
	return &TrafficConn{rawConn: conn}
}

func (this *TrafficConn) Read(b []byte) (n int, err error) {
	n, err = this.rawConn.Read(b)
	if n > 0 {
		atomic.AddUint64(&inTrafficBytes, uint64(n))
	}
	return
}

func (this *TrafficConn) Write(b []byte) (n int, err error) {
	n, err = this.rawConn.Write(b)
	if n > 0 {
		atomic.AddUint64(&outTrafficBytes, uint64(n))
	}
	return
}

func (this *TrafficConn) Close() error {
	return this.rawConn.Close()
}

func (this *TrafficConn) LocalAddr() net.Addr {
	return this.rawConn.LocalAddr()
}

func (this *TrafficConn) RemoteAddr() net.Addr {
	return this.rawConn.RemoteAddr()
}

func (this *TrafficConn) SetDeadline(t time.Time) error {
	return this.rawConn.SetDeadline(t)
}

func (this *TrafficConn) SetReadDeadline(t time.Time) error {
	return this.rawConn.SetReadDeadline(t)
}

func (this *TrafficConn) SetWriteDeadline(t time.Time) error {
	return this.rawConn.SetWriteDeadline(t)
}
