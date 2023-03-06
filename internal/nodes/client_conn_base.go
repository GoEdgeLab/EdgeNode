// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package nodes

import (
	"crypto/tls"
	"net"
)

type BaseClientConn struct {
	rawConn net.Conn

	isBound    bool
	userId     int64
	serverId   int64
	remoteAddr string
	hasLimit   bool

	isPersistent bool // 是否为持久化连接

	isClosed bool

	rawIP string
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
	this.hasLimit = true

	// 检查是否可以连接
	return sharedClientConnLimiter.Add(this.rawConn.RemoteAddr().String(), serverId, remoteAddr, maxConnsPerServer, maxConnsPerIP)
}

// SetServerId 设置服务ID
func (this *BaseClientConn) SetServerId(serverId int64) {
	this.serverId = serverId

	// 设置包装前连接
	switch conn := this.rawConn.(type) {
	case *tls.Conn:
		nativeConn, ok := conn.NetConn().(ClientConnInterface)
		if ok {
			nativeConn.SetServerId(serverId)
		}
	case *ClientConn:
		conn.SetServerId(serverId)
	}
}

// ServerId 读取当前连接绑定的服务ID
func (this *BaseClientConn) ServerId() int64 {
	return this.serverId
}

// SetUserId 设置所属服务的用户ID
func (this *BaseClientConn) SetUserId(userId int64) {
	this.userId = userId

	// 设置包装前连接
	switch conn := this.rawConn.(type) {
	case *tls.Conn:
		nativeConn, ok := conn.NetConn().(ClientConnInterface)
		if ok {
			nativeConn.SetUserId(userId)
		}
	case *ClientConn:
		conn.SetUserId(userId)
	}
}

// UserId 获取当前连接所属服务的用户ID
func (this *BaseClientConn) UserId() int64 {
	return this.userId
}

// RawIP 原本IP
func (this *BaseClientConn) RawIP() string {
	if len(this.rawIP) > 0 {
		return this.rawIP
	}

	ip, _, _ := net.SplitHostPort(this.rawConn.RemoteAddr().String())
	this.rawIP = ip
	return ip
}

// TCPConn 转换为TCPConn
func (this *BaseClientConn) TCPConn() (tcpConn *net.TCPConn, ok bool) {
	// 设置包装前连接
	switch conn := this.rawConn.(type) {
	case *tls.Conn:
		var internalConn = conn.NetConn()
		clientConn, ok := internalConn.(*ClientConn)
		if ok {
			return clientConn.TCPConn()
		}
		tcpConn, ok = internalConn.(*net.TCPConn)
	default:
		tcpConn, ok = this.rawConn.(*net.TCPConn)
	}
	return
}

// SetLinger 设置Linger
func (this *BaseClientConn) SetLinger(seconds int) error {
	tcpConn, ok := this.TCPConn()
	if ok {
		return tcpConn.SetLinger(seconds)
	}
	return nil
}

func (this *BaseClientConn) SetIsPersistent(isPersistent bool) {
	this.isPersistent = isPersistent
}
