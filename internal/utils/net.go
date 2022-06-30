//go:build !freebsd
// +build !freebsd

package utils

import (
	"context"
	"github.com/iwind/TeaGo/logs"
	"net"
	"syscall"
)

// ListenReuseAddr 监听可重用的端口
func ListenReuseAddr(network string, addr string) (net.Listener, error) {
	config := &net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			return c.Control(func(fd uintptr) {
				err := syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, SO_REUSEPORT, 1)
				if err != nil {
					logs.Println("[LISTEN]" + err.Error())
				}
			})
		},
		KeepAlive: 0,
	}
	return config.Listen(context.Background(), network, addr)
}

// ParseAddrHost 分析地址中的主机名部分
func ParseAddrHost(addr string) string {
	if len(addr) == 0 {
		return addr
	}

	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return host
}
