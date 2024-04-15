// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package conns_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/conns"
	"github.com/TeaOSLab/EdgeNode/internal/utils/testutils"
	"net"
	"runtime"
	"testing"
	"time"
)

type testConn struct {
	net.Conn

	addr net.Addr
}

func (this *testConn) Read(b []byte) (n int, err error) {
	return
}
func (this *testConn) Write(b []byte) (n int, err error) {
	return
}
func (this *testConn) Close() error {
	return nil
}
func (this *testConn) LocalAddr() net.Addr {
	return &net.TCPAddr{
		IP:   net.ParseIP(testutils.RandIP()),
		Port: 1234,
	}
}
func (this *testConn) RemoteAddr() net.Addr {
	if this.addr != nil {
		return this.addr
	}
	this.addr = &net.TCPAddr{
		IP:   net.ParseIP(testutils.RandIP()),
		Port: 1234,
	}
	return this.addr
}
func (this *testConn) SetDeadline(t time.Time) error {
	return nil
}
func (this *testConn) SetReadDeadline(t time.Time) error {
	return nil
}
func (this *testConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func BenchmarkMap_Add(b *testing.B) {
	runtime.GOMAXPROCS(512)

	var m = conns.NewMap()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var conn = &testConn{}
			m.Add(conn)
			m.Remove(conn)
		}
	})
}
