// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package nodes

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/firewallconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/firewalls"
	"github.com/TeaOSLab/EdgeNode/internal/iplibrary"
	"github.com/TeaOSLab/EdgeNode/internal/waf"
	"net"
	"time"
)

// ClientListener 客户端网络监听
type ClientListener struct {
	rawListener net.Listener
	isTLS       bool
	quickClose  bool
}

func NewClientListener(listener net.Listener, quickClose bool) *ClientListener {
	return &ClientListener{
		rawListener: listener,
		quickClose:  quickClose,
	}
}

func (this *ClientListener) SetIsTLS(isTLS bool) {
	this.isTLS = isTLS
}

func (this *ClientListener) IsTLS() bool {
	return this.isTLS
}

func (this *ClientListener) Accept() (net.Conn, error) {
	conn, err := this.rawListener.Accept()
	if err != nil {
		return nil, err
	}

	// 是否在WAF名单中
	ip, _, err := net.SplitHostPort(conn.RemoteAddr().String())
	if err == nil {
		canGoNext, _ := iplibrary.AllowIP(ip, 0)
		if !waf.SharedIPWhiteList.Contains(waf.IPTypeAll, firewallconfigs.FirewallScopeGlobal, 0, ip) {
			expiresAt, ok := waf.SharedIPBlackList.ContainsExpires(waf.IPTypeAll, firewallconfigs.FirewallScopeGlobal, 0, ip)
			if ok {
				var timeout = expiresAt - time.Now().Unix()
				if timeout > 0 {
					canGoNext = false

					if timeout > 3600 {
						timeout = 3600
					}

					// 使用本地防火墙延长封禁
					var fw = firewalls.Firewall()
					if fw != nil && !fw.IsMock() {
						// 这里 int(int64) 转换的前提是限制了 timeout <= 3600，否则将有整型溢出的风险
						_ = fw.DropSourceIP(ip, int(timeout), true)
					}
				}
			}
		}

		if !canGoNext {
			tcpConn, ok := conn.(*net.TCPConn)
			if ok {
				_ = tcpConn.SetLinger(0)
			}

			_ = conn.Close()

			return this.Accept()
		}
	}

	return NewClientConn(conn, this.isTLS, this.quickClose), nil
}

func (this *ClientListener) Close() error {
	return this.rawListener.Close()
}

func (this *ClientListener) Addr() net.Addr {
	return this.rawListener.Addr()
}
