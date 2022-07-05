// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package nodes

type ClientConnInterface interface {
	// IsClosed 是否已关闭
	IsClosed() bool

	// IsBound 是否已绑定服务
	IsBound() bool

	// Bind 绑定服务
	Bind(serverId int64, remoteAddr string, maxConnsPerServer int, maxConnsPerIP int) bool

	// ServerId 获取服务ID
	ServerId() int64

	// SetServerId 设置服务ID
	SetServerId(serverId int64)
}
