// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package nodes

import (
	"github.com/TeaOSLab/EdgeNode/internal/zero"
	"sync"
)

var sharedClientConnLimiter = NewClientConnLimiter()

// ClientConnRemoteAddr 客户端地址定义
type ClientConnRemoteAddr struct {
	remoteAddr string
	serverId   int64
}

// ClientConnLimiter 客户端连接数限制
type ClientConnLimiter struct {
	remoteAddrMap map[string]*ClientConnRemoteAddr // raw remote addr => remoteAddr
	ipConns       map[string]map[string]zero.Zero  // remoteAddr => { raw remote addr => Zero }
	serverConns   map[int64]map[string]zero.Zero   // serverId => { remoteAddr => Zero }

	locker sync.Mutex
}

func NewClientConnLimiter() *ClientConnLimiter {
	return &ClientConnLimiter{
		remoteAddrMap: map[string]*ClientConnRemoteAddr{},
		ipConns:       map[string]map[string]zero.Zero{},
		serverConns:   map[int64]map[string]zero.Zero{},
	}
}

// Add 添加新连接
// 返回值为true的时候表示允许添加；否则表示不允许添加
func (this *ClientConnLimiter) Add(rawRemoteAddr string, serverId int64, remoteAddr string, maxConnsPerServer int, maxConnsPerIP int) bool {
	if (maxConnsPerServer <= 0 && maxConnsPerIP <= 0) || len(remoteAddr) == 0 || serverId <= 0 {
		return true
	}

	this.locker.Lock()
	defer this.locker.Unlock()

	// 检查服务连接数
	var serverMap = this.serverConns[serverId]
	if maxConnsPerServer > 0 {
		if serverMap == nil {
			serverMap = map[string]zero.Zero{}
			this.serverConns[serverId] = serverMap
		}

		if maxConnsPerServer <= len(serverMap) {
			return false
		}
	}

	// 检查IP连接数
	var ipMap = this.ipConns[remoteAddr]
	if maxConnsPerIP > 0 {
		if ipMap == nil {
			ipMap = map[string]zero.Zero{}
			this.ipConns[remoteAddr] = ipMap
		}
		if maxConnsPerIP > 0 && maxConnsPerIP <= len(ipMap) {
			return false
		}
	}

	this.remoteAddrMap[rawRemoteAddr] = &ClientConnRemoteAddr{
		remoteAddr: remoteAddr,
		serverId:   serverId,
	}

	if maxConnsPerServer > 0 {
		serverMap[rawRemoteAddr] = zero.New()
	}
	if maxConnsPerIP > 0 {
		ipMap[rawRemoteAddr] = zero.New()
	}

	return true
}

// Remove 删除连接
func (this *ClientConnLimiter) Remove(rawRemoteAddr string) {
	this.locker.Lock()
	defer this.locker.Unlock()

	addr, ok := this.remoteAddrMap[rawRemoteAddr]
	if !ok {
		return
	}

	delete(this.remoteAddrMap, rawRemoteAddr)
	delete(this.ipConns[addr.remoteAddr], rawRemoteAddr)
	delete(this.serverConns[addr.serverId], rawRemoteAddr)

	if len(this.ipConns[addr.remoteAddr]) == 0 {
		delete(this.ipConns, addr.remoteAddr)
	}

	if len(this.serverConns[addr.serverId]) == 0 {
		delete(this.serverConns, addr.serverId)
	}
}

// Conns 获取连接信息
// 用于调试
func (this *ClientConnLimiter) Conns() (ipConns map[string][]string, serverConns map[int64][]string) {
	this.locker.Lock()
	defer this.locker.Unlock()

	ipConns = map[string][]string{}    // ip => [addr1, addr2, ...]
	serverConns = map[int64][]string{} // serverId => [addr1, addr2, ...]

	for ip, m := range this.ipConns {
		for addr := range m {
			ipConns[ip] = append(ipConns[ip], addr)
		}
	}

	for serverId, m := range this.serverConns {
		for addr := range m {
			serverConns[serverId] = append(serverConns[serverId], addr)
		}
	}

	return
}
