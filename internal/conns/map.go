// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package conns

import (
	"github.com/iwind/TeaGo/types"
	"net"
	"sync"
)

var SharedMap = NewMap()

type Map struct {
	m map[string]map[string]net.Conn // ip => { network_port => Conn  }

	locker sync.RWMutex
}

func NewMap() *Map {
	return &Map{
		m: map[string]map[string]net.Conn{},
	}
}

func (this *Map) Add(conn net.Conn) {
	if conn == nil {
		return
	}

	key, ip, ok := this.connAddr(conn)
	if !ok {
		return
	}

	this.locker.Lock()
	defer this.locker.Unlock()
	connMap, ok := this.m[ip]
	if !ok {
		this.m[ip] = map[string]net.Conn{key: conn}
	} else {
		connMap[key] = conn
	}
}

func (this *Map) Remove(conn net.Conn) {
	if conn == nil {
		return
	}
	key, ip, ok := this.connAddr(conn)
	if !ok {
		return
	}

	this.locker.Lock()
	defer this.locker.Unlock()

	connMap, ok := this.m[ip]
	if !ok {
		return
	}
	delete(connMap, key)

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

func (this *Map) AllConns() []net.Conn {
	this.locker.RLock()
	defer this.locker.RUnlock()

	var result = []net.Conn{}
	for _, m := range this.m {
		for _, connInfo := range m {
			result = append(result, connInfo)
		}
	}

	return result
}

func (this *Map) connAddr(conn net.Conn) (key string, ip string, ok bool) {
	if conn == nil {
		return
	}

	var addr = conn.RemoteAddr()
	switch realAddr := addr.(type) {
	case *net.TCPAddr:
		return addr.Network() + types.String(realAddr.Port), realAddr.IP.String(), true
	case *net.UDPAddr:
		return addr.Network() + types.String(realAddr.Port), realAddr.IP.String(), true
	default:
		var s = addr.String()
		host, port, err := net.SplitHostPort(s)
		if err != nil {
			return
		}
		return addr.Network() + port, host, true
	}
}
