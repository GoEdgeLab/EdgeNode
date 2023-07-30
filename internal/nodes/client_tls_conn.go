// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package nodes

import (
	"crypto/tls"
	"net"
	"time"
)

// ClientTLSConn TLS连接封装
type ClientTLSConn struct {
	BaseClientConn
}

func NewClientTLSConn(conn *tls.Conn) net.Conn {
	return &ClientTLSConn{BaseClientConn{rawConn: conn}}
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

	// 单个服务并发数限制
	sharedClientConnLimiter.Remove(this.rawConn.RemoteAddr().String())

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

func (this *ClientTLSConn) SetIsPersistent(isPersistent bool) {
	tlsConn, ok := this.rawConn.(*tls.Conn)
	if ok {
		var rawConn = tlsConn.NetConn()
		if rawConn != nil {
			clientConn, ok := rawConn.(*ClientConn)
			if ok {
				clientConn.SetIsPersistent(isPersistent)
			}
		}
	}
}

func (this *ClientTLSConn) Fingerprint() []byte {
	tlsConn, ok := this.rawConn.(*tls.Conn)
	if ok {
		var rawConn = tlsConn.NetConn()
		if rawConn != nil {
			clientConn, ok := rawConn.(*ClientConn)
			if ok {
				return clientConn.fingerprint
			}
		}
	}
	return nil
}

// LastRequestBytes 读取上一次请求发送的字节数
func (this *ClientTLSConn) LastRequestBytes() int64 {
	tlsConn, ok := this.rawConn.(*tls.Conn)
	if ok {
		var rawConn = tlsConn.NetConn()
		if rawConn != nil {
			clientConn, ok := rawConn.(*ClientConn)
			if ok {
				return clientConn.LastRequestBytes()
			}
		}
	}
	return 0
}
