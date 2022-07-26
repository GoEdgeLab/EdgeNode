// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package teaconst

import "github.com/TeaOSLab/EdgeCommon/pkg/nodeconfigs"

var (
	// 流量统计

	InTrafficBytes  = uint64(0)
	OutTrafficBytes = uint64(0)

	NodeId       int64 = 0
	NodeIdString       = ""
	IsDaemon           = false

	GlobalProductName = nodeconfigs.DefaultProductName

	IsQuiting    = false // 是否正在退出
	EnableDBStat = false // 是否开启本地数据库统计

	DiskIsFast = false // 是否为高速硬盘
)
