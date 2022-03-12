// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package utils

import (
	"github.com/shirou/gopsutil/v3/mem"
)

var systemTotalMemory = -1

func init() {
	_ = SystemMemoryGB()
}

func SystemMemoryGB() int {
	if systemTotalMemory > 0 {
		return systemTotalMemory
	}

	stat, err := mem.VirtualMemory()
	if err != nil {
		return 0
	}

	systemTotalMemory = int(stat.Total / 1024 / 1024 / 1024)
	return systemTotalMemory
}
