// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package nodes

import "net"

type BaseClientConn struct {
	rawConn net.Conn

	isBound    bool
	serverId   int64
	remoteAddr string

	isClosed bool
}

func (this *BaseClientConn) IsClosed() bool {
	return this.isClosed
}

// IsBound 是否已绑定服务
func (this *BaseClientConn) IsBound() bool {
	return this.isBound
}

// Bind 绑定服务
func (this *BaseClientConn) Bind(serverId int64, remoteAddr string, maxConnsPerServer int, maxConnsPerIP int) bool {
	if this.isBound {
		return true
	}
	this.isBound = true
	this.serverId = serverId
	this.remoteAddr = remoteAddr

	// 检查是否可以连接
	return sharedClientConnLimiter.Add(this.rawConn.RemoteAddr().String(), serverId, remoteAddr, maxConnsPerServer, maxConnsPerIP)
}

