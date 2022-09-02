// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package conns

import (
	"net"
	"sync"
)

var SharedMap = NewMap()

type Map struct {
	m map[string]map[int]net.Conn // ip => { port => Conn }

	locker sync.RWMutex
}

func NewMap() *Map {
	return &Map{
		m: map[string]map[int]net.Conn{},
	}
}

func (this *Map) Add(conn net.Conn) {
	if conn == nil {
		return
	}
	tcpAddr, ok := conn.RemoteAddr().(*net.TCPAddr)
	if !ok {
		return
	}

	var ip = tcpAddr.IP.String()
	var port = tcpAddr.Port

	this.locker.Lock()
	defer this.locker.Unlock()
	connMap, ok := this.m[ip]
	if !ok {
		this.m[ip] = map[int]net.Conn{
			port: conn,
		}
	} else {
		connMap[port] = conn
	}
}

func (this *Map) Remove(conn net.Conn) {
	if conn == nil {
		return
	}
	tcpAddr, ok := conn.RemoteAddr().(*net.TCPAddr)
	if !ok {
		return
	}

	var ip = tcpAddr.IP.String()
	var port = tcpAddr.Port

	this.locker.Lock()
	defer this.locker.Unlock()

	connMap, ok := this.m[ip]
	if !ok {
		return
	}
	delete(connMap, port)

	if len(connMap) == 0 {
		delete(this.m, ip)
	}
}

func (this *Map) CountIPConns(ip string) int {
	this.locker.RLock()
	var l = len(this.m[ip])
	this.locker.RUnlock()
	return l
}

func (this *Map) CloseIPConns(ip string) {
	var conns = []net.Conn{}

	this.locker.RLock()
	connMap, ok := this.m[ip]

	// 复制，防止在Close时产生并发冲突
	if ok {
		for _, conn := range connMap {
			conns = append(conns, conn)
		}
	}

	// 需要在Close之前结束，防止死循环
	this.locker.RUnlock()

	if ok {
		for _, conn := range conns {
			_ = conn.Close()
		}

		// 这里不需要从 m 中删除，因为关闭时会自然触发回调
	}
}

func (this *Map) AllConns() []net.Conn {
	this.locker.RLock()
	defer this.locker.RUnlock()

	var result = []net.Conn{}
	for _, m := range this.m {
		for _, conn := range m {
			result = append(result, conn)
		}
	}
	return result
}
