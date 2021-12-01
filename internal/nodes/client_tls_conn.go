// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package nodes

import (
	"crypto/tls"
	"net"
	"time"
)

type ClientTLSConn struct {
	rawConn  *tls.Conn
	isClosed bool
}

func NewClientTLSConn(conn *tls.Conn) net.Conn {
	return &ClientTLSConn{rawConn: conn}
}

func (this *ClientTLSConn) Read(b []byte) (n int, err error) {
	n, err = this.rawConn.Read(b)
	return
}

func (this *ClientTLSConn) Write(b []byte) (n int, err error) {
	n, err = this.rawConn.Write(b)
	return
}

func (this *ClientTLSConn) Close() error {
	this.isClosed = true
	return this.rawConn.Close()
}

func (this *ClientTLSConn) LocalAddr() net.Addr {
	return this.rawConn.LocalAddr()
}

func (this *ClientTLSConn) RemoteAddr() net.Addr {
	return this.rawConn.RemoteAddr()
}

func (this *ClientTLSConn) SetDeadline(t time.Time) error {
	return this.rawConn.SetDeadline(t)
}

func (this *ClientTLSConn) SetReadDeadline(t time.Time) error {
	return this.rawConn.SetReadDeadline(t)
}

func (this *ClientTLSConn) SetWriteDeadline(t time.Time) error {
	return this.rawConn.SetWriteDeadline(t)
}

func (this *ClientTLSConn) IsClosed() bool {
	return this.isClosed
}
