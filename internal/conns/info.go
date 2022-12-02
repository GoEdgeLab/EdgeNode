// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package conns

import "net"

type ConnInfo struct {
	Conn      net.Conn
	CreatedAt int64
}
