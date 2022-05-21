// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package nodes

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/firewallconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/firewalls"
	"github.com/TeaOSLab/EdgeNode/internal/iplibrary"
	"github.com/TeaOSLab/EdgeNode/internal/waf"
	"net"
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

		if !canGoNext ||
			(!waf.SharedIPWhiteList.Contains(waf.IPTypeAll, firewallconfigs.FirewallScopeGlobal, 0, ip) &&
				waf.SharedIPBlackList.Contains(waf.IPTypeAll, firewallconfigs.FirewallScopeGlobal, 0, ip)) {
			tcpConn, ok := conn.(*net.TCPConn)
			if ok {
				_ = tcpConn.SetLinger(0)
			}

			_ = conn.Close()

			// 使用本地防火墙延长封禁
			var fw = firewalls.Firewall()
			if fw != nil && !fw.IsMock() {
				_ = fw.DropSourceIP(ip, 60)
			}

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
