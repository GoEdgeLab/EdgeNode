// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package teaconst

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/nodeconfigs"
	"os"
	"strings"
)

var (
	// 流量统计

	InTrafficBytes  = uint64(0)
	OutTrafficBytes = uint64(0)

	NodeId       int64 = 0
	NodeIdString       = ""
	IsMain             = checkMain()

	GlobalProductName = nodeconfigs.DefaultProductName

	IsQuiting    = false // 是否正在退出
	EnableDBStat = false // 是否开启本地数据库统计
)

// 检查是否为主程序
func checkMain() bool {
	if len(os.Args) == 1 ||
		(len(os.Args) >= 2 && os.Args[1] == "pprof") {
		return true
	}
	exe, _ := os.Executable()
	return strings.HasSuffix(exe, ".test") ||
		strings.HasSuffix(exe, ".test.exe") ||
		strings.Contains(exe, "___")
}
