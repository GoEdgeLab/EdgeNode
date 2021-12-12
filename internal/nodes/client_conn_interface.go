// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package nodes

type ClientConnInterface interface {
	// IsClosed 是否已关闭
	IsClosed() bool

	// IsBound 是否已绑定服务
	IsBound() bool

	// Bind 绑定服务
	Bind(serverId int64, remoteAddr string, maxConnsPerServer int, maxConnsPerIP int) bool
}
