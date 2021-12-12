// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package nodes

import (
	"net"
)

// 判断客户端连接是否已关闭
func isClientConnClosed(conn net.Conn) bool {
	if conn == nil {
		return true
	}
	clientConn, ok := conn.(ClientConnInterface)
	if ok {
		return clientConn.IsClosed()
	}

	return true
}
