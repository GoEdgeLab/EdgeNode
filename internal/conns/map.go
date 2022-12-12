// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package conns

import (
	"net"
	"sync"
	"time"
)

var SharedMap = NewMap()

type Map struct {
	m map[string]map[int]*ConnInfo // ip => { port => ConnInfo  }

	locker sync.RWMutex
}

func NewMap() *Map {
	return &Map{
		m: map[string]map[int]*ConnInfo{},
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

	var connInfo = &ConnInfo{
		Conn:      conn,
		CreatedAt: time.Now().Unix(),
	}

	this.locker.Lock()
	defer this.locker.Unlock()
	connMap, ok := this.m[ip]
	if !ok {
		this.m[ip] = map[int]*ConnInfo{
			port: connInfo,
		}
	} else {
		connMap[port] = connInfo
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
		for _, connInfo := range connMap {
			conns = append(conns, connInfo.Conn)
		}
	}

	// 需要在Close之前结束，防止死循环
	this.locker.RUnlock()

	if ok {
		for _, conn := range conns {
			// 设置Linger
			lingerConn, isLingerConn := conn.(LingerConn)
			if isLingerConn {
				_ = lingerConn.SetLinger(0)
			}

			// 关闭
			_ = conn.Close()
		}

		// 这里不需要从 m 中删除，因为关闭时会自然触发回调
	}
}

func (this *Map) AllConns() []*ConnInfo {
	this.locker.RLock()
	defer this.locker.RUnlock()

	var result = []*ConnInfo{}
	for _, m := range this.m {
		for _, connInfo := range m {
			result = append(result, connInfo)
		}
	}
	return result
}
