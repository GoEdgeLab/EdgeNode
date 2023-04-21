// Copyright 2023 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package connutils

import (
	"github.com/TeaOSLab/EdgeNode/internal/zero"
	"net"
	"sync"
	"time"
)

// 记录不需要带宽统计的连接
// 比如本地的清理和预热
var noStatAddrMap = map[string]zero.Zero{} // addr => Zero
var noStatLocker = &sync.RWMutex{}

// IsNoStatConn 检查是否为不统计连接
func IsNoStatConn(addr string) bool {
	noStatLocker.RLock()
	_, ok := noStatAddrMap[addr]
	noStatLocker.RUnlock()
	return ok
}

type NoStatConn struct {
	rawConn net.Conn
}

func NewNoStat(rawConn net.Conn) net.Conn {
	noStatLocker.Lock()
	noStatAddrMap[rawConn.LocalAddr().String()] = zero.New()
	noStatLocker.Unlock()
	return &NoStatConn{rawConn: rawConn}
}

func (this *NoStatConn) Read(b []byte) (n int, err error) {
	return this.rawConn.Read(b)
}

func (this *NoStatConn) Write(b []byte) (n int, err error) {
	return this.rawConn.Write(b)
}

func (this *NoStatConn) Close() error {
	err := this.rawConn.Close()

	noStatLocker.Lock()
	delete(noStatAddrMap, this.rawConn.LocalAddr().String())
	noStatLocker.Unlock()

	return err
}

func (this *NoStatConn) LocalAddr() net.Addr {
	return this.rawConn.LocalAddr()
}

func (this *NoStatConn) RemoteAddr() net.Addr {
	return this.rawConn.RemoteAddr()
}

func (this *NoStatConn) SetDeadline(t time.Time) error {
	return this.rawConn.SetDeadline(t)
}

func (this *NoStatConn) SetReadDeadline(t time.Time) error {
	return this.rawConn.SetReadDeadline(t)
}

func (this *NoStatConn) SetWriteDeadline(t time.Time) error {
	return this.rawConn.SetWriteDeadline(t)
}
