// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package nodes

import "net"

// TrafficListener 用于统计流量的网络监听
type TrafficListener struct {
	rawListener net.Listener
}

func NewTrafficListener(listener net.Listener) net.Listener {
	return &TrafficListener{rawListener: listener}
}

func (this *TrafficListener) Accept() (net.Conn, error) {
	conn, err := this.rawListener.Accept()
	if err != nil {
		return nil, err
	}
	return NewTrafficConn(conn), nil
}

func (this *TrafficListener) Close() error {
	return this.rawListener.Close()
}

func (this *TrafficListener) Addr() net.Addr {
	return this.rawListener.Addr()
}
