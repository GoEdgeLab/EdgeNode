// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package nodes

import (
	"crypto/tls"
	"github.com/TeaOSLab/EdgeNode/internal/firewalls"
	"github.com/TeaOSLab/EdgeNode/internal/iplibrary"
	"net"
	"sync/atomic"
	"time"
)

type BaseClientConn struct {
	rawConn net.Conn

	isBound    bool
	userId     int64
	userPlanId int64
	serverId   int64
	remoteAddr string
	hasLimit   bool

	isPersistent bool // 是否为持久化连接
	fingerprint  []byte

	isClosed bool

	rawIP string

	totalSentBytes int64
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
func (this *BaseClientConn) SetServerId(serverId int64) (goNext bool) {
	goNext = true

	// 检查服务相关IP黑名单
	var rawIP = this.RawIP()
	if serverId > 0 && len(rawIP) > 0 {
		// 是否在白名单中
		ok, _, expiresAt := iplibrary.AllowIP(rawIP, serverId)
		if !ok {
			_ = this.rawConn.Close()
			firewalls.DropTemporaryTo(rawIP, expiresAt)
			return false
		}
	}

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

	return true
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

func (this *BaseClientConn) SetUserPlanId(userPlanId int64) {
	this.userPlanId = userPlanId

	// 设置包装前连接
	switch conn := this.rawConn.(type) {
	case *tls.Conn:
		nativeConn, ok := conn.NetConn().(ClientConnInterface)
		if ok {
			nativeConn.SetUserPlanId(userPlanId)
		}
	case *ClientConn:
		conn.SetUserPlanId(userPlanId)
	}
}

// UserId 获取当前连接所属服务的用户ID
func (this *BaseClientConn) UserId() int64 {
	return this.userId
}

// UserPlanId 用户套餐ID
func (this *BaseClientConn) UserPlanId() int64 {
	return this.userPlanId
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
		clientConn, isClientConn := internalConn.(*ClientConn)
		if isClientConn {
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

	_ = this.rawConn.SetDeadline(time.Time{})
}

// SetFingerprint 设置指纹信息
func (this *BaseClientConn) SetFingerprint(fingerprint []byte) {
	this.fingerprint = fingerprint
}

// Fingerprint 读取指纹信息
func (this *BaseClientConn) Fingerprint() []byte {
	return this.fingerprint
}

// LastRequestBytes 读取上一次请求发送的字节数
func (this *BaseClientConn) LastRequestBytes() int64 {
	var result = atomic.LoadInt64(&this.totalSentBytes)
	atomic.StoreInt64(&this.totalSentBytes, 0)
	return result
}
