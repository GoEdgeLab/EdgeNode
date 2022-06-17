// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package nodes

type LingerConn interface {
	SetLinger(sec int) error
}
