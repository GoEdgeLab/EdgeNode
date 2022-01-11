// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package nodes

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/nodeconfigs"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/firewallconfigs"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/iplibrary"
	"github.com/TeaOSLab/EdgeNode/internal/ratelimit"
	"github.com/TeaOSLab/EdgeNode/internal/ttlcache"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/TeaOSLab/EdgeNode/internal/waf"
	"github.com/iwind/TeaGo/types"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// ClientConn 客户端连接
type ClientConn struct {
	once          sync.Once
	globalLimiter *ratelimit.Counter

	isTLS       bool
	hasDeadline bool
	hasRead     bool

	BaseClientConn
}

func NewClientConn(conn net.Conn, isTLS bool, quickClose bool, globalLimiter *ratelimit.Counter) net.Conn {
	if quickClose {
		// TCP
		tcpConn, ok := conn.(*net.TCPConn)
		if ok {
			// TODO 可以在配置中设置此值
			_ = tcpConn.SetLinger(nodeconfigs.DefaultTCPLinger)
		}
	}

	return &ClientConn{BaseClientConn: BaseClientConn{rawConn: conn}, isTLS: isTLS, globalLimiter: globalLimiter}
}

func (this *ClientConn) Read(b []byte) (n int, err error) {
	if this.isTLS {
		if !this.hasDeadline {
			_ = this.rawConn.SetReadDeadline(time.Now().Add(time.Duration(nodeconfigs.DefaultTLSHandshakeTimeout) * time.Second)) // TODO 握手超时时间可以设置
			this.hasDeadline = true
			defer func() {
				_ = this.rawConn.SetReadDeadline(time.Time{})
			}()
		}
	}

	n, err = this.rawConn.Read(b)
	if n > 0 {
		atomic.AddUint64(&teaconst.InTrafficBytes, uint64(n))
		this.hasRead = true
	}

	// SYN Flood检测
	var synFloodConfig = sharedNodeConfig.SYNFloodConfig()
	if synFloodConfig != nil && synFloodConfig.IsOn {
		if err != nil && os.IsTimeout(err) {
			if !this.hasRead {
				this.checkSYNFlood(synFloodConfig)
			}
		} else if err == nil {
			this.resetSYNFlood()
		}
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

	err := this.rawConn.Close()

	// 全局并发数限制
	this.once.Do(func() {
		if this.globalLimiter != nil {
			this.globalLimiter.Release()
		}
	})

	// 单个服务并发数限制
	sharedClientConnLimiter.Remove(this.rawConn.RemoteAddr().String())

	return err
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

func (this *ClientConn) resetSYNFlood() {
	//ttlcache.SharedCache.Delete("SYN_FLOOD:" + this.RawIP())
}

func (this *ClientConn) checkSYNFlood(synFloodConfig *firewallconfigs.SYNFloodConfig) {
	var ip = this.RawIP()
	if len(ip) > 0 && !iplibrary.IsInWhiteList(ip) && (!synFloodConfig.IgnoreLocal || !utils.IsLocalIP(ip)) {
		var timestamp = utils.NextMinuteUnixTime()
		var result = ttlcache.SharedCache.IncreaseInt64("SYN_FLOOD:"+ip, 1, timestamp)
		var minAttempts = synFloodConfig.MinAttempts
		if minAttempts < 5 {
			minAttempts = 5
		}
		if result >= int64(minAttempts) {
			var timeout = synFloodConfig.TimeoutSeconds
			if timeout <= 0 {
				timeout = 600
			}
			waf.SharedIPBlackList.RecordIP(waf.IPTypeAll, firewallconfigs.FirewallScopeGlobal, 0, ip, time.Now().Unix()+int64(timeout), 0, true, 0, 0, "疑似SYN Flood攻击，当前1分钟"+types.String(result)+"次空连接")
		}
	}
}
