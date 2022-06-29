// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package nodes

import "github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"

type OriginState struct {
	CountFails   int64
	UpdatedAt    int64
	Config       *serverconfigs.OriginConfig
	Addr         string
	TLSHost      string
	ReverseProxy *serverconfigs.ReverseProxyConfig
}
