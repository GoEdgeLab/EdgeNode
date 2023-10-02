// Copyright 2023 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .
//go:build go1.19

package utils

import (
	"runtime/debug"
)

// 设置软内存最大值
func setMaxMemory(memoryGB int) {
	if memoryGB <= 0 {
		memoryGB = 1
	}

	var maxMemoryBytes = (int64(memoryGB) << 30) * 75 / 100 // 默认 75%
	debug.SetMemoryLimit(maxMemoryBytes)
}
