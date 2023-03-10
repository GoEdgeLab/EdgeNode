// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package utils

import (
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/shirou/gopsutil/v3/mem"
)

var systemTotalMemory = -1

func init() {
	if !teaconst.IsMain {
		return
	}

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
	if systemTotalMemory <= 0 {
		systemTotalMemory = 1
	}

	setMaxMemory(systemTotalMemory)

	return systemTotalMemory
}
