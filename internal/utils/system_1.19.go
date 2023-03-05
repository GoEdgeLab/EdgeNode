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
	var maxMemoryBytes int64 = 0
	if memoryGB > 10 {
		maxMemoryBytes = int64(memoryGB-2) << 30 // 超过10G内存的允许剩余2G内存
	} else {
		maxMemoryBytes = (int64(memoryGB) << 30) * 80 / 100 // 默认 80%
	}

	debug.SetMemoryLimit(maxMemoryBytes)
}
