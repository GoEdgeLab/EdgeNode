// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package utils

import (
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/shirou/gopsutil/v3/mem"
)

var systemTotalMemory = -1
var systemMemoryBytes uint64

func init() {
	if !teaconst.IsMain {
		return
	}

	_ = SystemMemoryGB()
}

// SystemMemoryGB 系统内存GB数量
// 必须保证不小于1
func SystemMemoryGB() int {
	if systemTotalMemory > 0 {
		return systemTotalMemory
	}

	stat, err := mem.VirtualMemory()
	if err != nil {
		return 1
	}

	systemMemoryBytes = stat.Total

	systemTotalMemory = int(stat.Total / (1 << 30))
	if systemTotalMemory <= 0 {
		systemTotalMemory = 1
	}

	setMaxMemory(systemTotalMemory)

	return systemTotalMemory
}

// SystemMemoryBytes 系统内存总字节数
func SystemMemoryBytes() uint64 {
	return systemMemoryBytes
}
