// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package ttlcache

import (
	"github.com/shirou/gopsutil/mem"
)

var systemTotalMemory = -1

func systemMemoryGB() int {
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
