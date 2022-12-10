// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package nodes

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/nodeconfigs"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/firewallconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/conns"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/iplibrary"
	"github.com/TeaOSLab/EdgeNode/internal/stats"
	"github.com/TeaOSLab/EdgeNode/internal/ttlcache"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/TeaOSLab/EdgeNode/internal/waf"
	"github.com/iwind/TeaGo/Tea"
	"github.com/iwind/TeaGo/types"
	"net"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

// ClientConn 客户端连接
type ClientConn struct {
	BaseClientConn

	isTLS       bool
	hasDeadline bool
	hasRead     bool

	isLO          bool // 是否为环路
	isInAllowList bool

	hasResetSYNFlood bool
}

func NewClientConn(rawConn net.Conn, isTLS bool, quickClose bool, isInAllowList bool) net.Conn {
	// 是否为环路
	var remoteAddr = rawConn.RemoteAddr().String()
	var isLO = strings.HasPrefix(remoteAddr, "127.0.0.1:") || strings.HasPrefix(remoteAddr, "[::1]:")

	var conn = &ClientConn{
		BaseClientConn: BaseClientConn{rawConn: rawConn},
		isTLS:          isTLS,
		isLO:           isLO,
		isInAllowList:  isInAllowList,
	}

	if quickClose {
		// TODO 可以在配置中设置此值
		_ = conn.SetLinger(nodeconfigs.DefaultTCPLinger)
	}

	// 加入到Map
	conns.SharedMap.Add(conn)

	return conn
}

func (this *ClientConn) Read(b []byte) (n int, err error) {
	// 环路直接读取
	if this.isLO {
		n, err = this.rawConn.Read(b)
		if n > 0 {
			atomic.AddUint64(&teaconst.InTrafficBytes, uint64(n))
		}
		return
	}

	// TLS
	if this.isTLS {
		if !this.hasDeadline {
			_ = this.rawConn.SetReadDeadline(time.Now().Add(time.Duration(nodeconfigs.DefaultTLSHandshakeTimeout) * time.Second)) // TODO 握手超时时间可以设置
			this.hasDeadline = true
			defer func() {
				_ = this.rawConn.SetReadDeadline(time.Time{})
			}()
		}
	}

	// 开始读取
	n, err = this.rawConn.Read(b)
	if n > 0 {
		atomic.AddUint64(&teaconst.InTrafficBytes, uint64(n))
		if !this.hasRead {
			this.hasRead = true
		}
	}

	// 检测是否为握手错误
	var isHandshakeError = err != nil && os.IsTimeout(err) && !this.hasRead
	if isHandshakeError {
		_ = this.SetLinger(0)
	}

	// 忽略白名单和局域网
	if !this.isInAllowList && !utils.IsLocalIP(this.RawIP()) {
		// SYN Flood检测
		if this.serverId == 0 || !this.hasResetSYNFlood {
			var synFloodConfig = sharedNodeConfig.SYNFloodConfig()
			if synFloodConfig != nil && synFloodConfig.IsOn {
				if isHandshakeError {
					this.increaseSYNFlood(synFloodConfig)
				} else if err == nil && !this.hasResetSYNFlood {
					this.hasResetSYNFlood = true
					this.resetSYNFlood()
				}
			}
		}
	}

	return
}

func (this *ClientConn) Write(b []byte) (n int, err error) {
	// 设置超时时间
	_ = this.rawConn.SetWriteDeadline(time.Now().Add(60 * time.Second)) // TODO 时间可以设置

	n, err = this.rawConn.Write(b)
	if n > 0 {
		// 统计当前服务带宽
		if this.serverId > 0 {
			if !this.isLO || Tea.IsTesting() { // 环路不统计带宽，避免缓存预热等行为产生带宽
				atomic.AddUint64(&teaconst.OutTrafficBytes, uint64(n))
				stats.SharedBandwidthStatManager.Add(this.userId, this.serverId, int64(n))
			}
		}
	}

	// 如果是写入超时，则立即关闭连接
	if err != nil && os.IsTimeout(err) {
		conn, ok := this.rawConn.(LingerConn)
		if ok {
			_ = conn.SetLinger(0)
		}
	}

	return
}

func (this *ClientConn) Close() error {
	this.isClosed = true

	err := this.rawConn.Close()

	// 单个服务并发数限制
	// 不能加条件限制，因为服务配置随时有变化
	sharedClientConnLimiter.Remove(this.rawConn.RemoteAddr().String())

	// 从conn map中移除
	conns.SharedMap.Remove(this)

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
	ttlcache.SharedCache.Delete("SYN_FLOOD:" + this.RawIP())
}

func (this *ClientConn) increaseSYNFlood(synFloodConfig *firewallconfigs.SYNFloodConfig) {
	var ip = this.RawIP()
	if len(ip) > 0 && !iplibrary.IsInWhiteList(ip) && (!synFloodConfig.IgnoreLocal || !utils.IsLocalIP(ip)) {
		var timestamp = utils.NextMinuteUnixTime()
		var result = ttlcache.SharedCache.IncreaseInt64("SYN_FLOOD:"+ip, 1, timestamp, true)
		var minAttempts = synFloodConfig.MinAttempts
		if minAttempts < 5 {
			minAttempts = 5
		}
		if !this.isTLS {
			// 非TLS，设置为两倍，防止误封
			minAttempts = 2 * minAttempts
		}
		if result >= int64(minAttempts) {
			var timeout = synFloodConfig.TimeoutSeconds
			if timeout <= 0 {
				timeout = 600
			}

			// 关闭当前连接
			_ = this.SetLinger(0)
			_ = this.Close()

			waf.SharedIPBlackList.RecordIP(waf.IPTypeAll, firewallconfigs.FirewallScopeGlobal, 0, ip, time.Now().Unix()+int64(timeout), 0, true, 0, 0, "疑似SYN Flood攻击，当前1分钟"+types.String(result)+"次空连接")
		}
	}
}
