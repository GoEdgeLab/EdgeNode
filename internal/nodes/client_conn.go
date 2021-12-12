// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package nodes

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/nodeconfigs"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/ratelimit"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// ClientConn 客户端连接
type ClientConn struct {
	once          sync.Once
	globalLimiter *ratelimit.Counter

	BaseClientConn
}

func NewClientConn(conn net.Conn, quickClose bool, globalLimiter *ratelimit.Counter) net.Conn {
	if quickClose {
		// TCP
		tcpConn, ok := conn.(*net.TCPConn)
		if ok {
			// TODO 可以在配置中设置此值
			_ = tcpConn.SetLinger(nodeconfigs.DefaultTCPLinger)
		}
	}

	return &ClientConn{BaseClientConn: BaseClientConn{rawConn: conn}, globalLimiter: globalLimiter}
}

func (this *ClientConn) Read(b []byte) (n int, err error) {
	n, err = this.rawConn.Read(b)
	if n > 0 {
		atomic.AddUint64(&teaconst.InTrafficBytes, uint64(n))
	}
	return
}

func (this *ClientConn) Write(b []byte) (n int, err error) {
	n, err = this.rawConn.Write(b)
	if n > 0 {
		atomic.AddUint64(&teaconst.OutTrafficBytes, uint64(n))
	}
	return
}

func (this *ClientConn) Close() error {
	this.isClosed = true

	// 全局并发数限制
	this.once.Do(func() {
		if this.globalLimiter != nil {
			this.globalLimiter.Release()
		}
	})

	// 单个服务并发数限制
	sharedClientConnLimiter.Remove(this.rawConn.RemoteAddr().String())

	return this.rawConn.Close()
}

func (this *ClientConn) LocalAddr() net.Addr {
	return this.rawConn.LocalAddr()
}

func (this *ClientConn) RemoteAddr() net.Addr {
	return this.rawConn.RemoteAddr()
}

func (this *ClientConn) SetDeadline(t time.Time) error {
	return this.rawConn.SetDeadline(t)
}

func (this *ClientConn) SetReadDeadline(t time.Time) error {
	return this.rawConn.SetReadDeadline(t)
}

func (this *ClientConn) SetWriteDeadline(t time.Time) error {
	return this.rawConn.SetWriteDeadline(t)
}
