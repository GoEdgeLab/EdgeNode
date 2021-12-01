// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package nodes

type ClientConnCloser interface {
	IsClosed() bool
}
