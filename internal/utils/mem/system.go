// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package memutils

import (
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/shirou/gopsutil/v3/mem"
	"time"
)

var systemTotalMemory = -1
var systemMemoryBytes uint64
var availableMemoryGB int

func init() {
	if !teaconst.IsMain {
		return
	}

	_ = SystemMemoryGB()

	goman.New(func() {
		var ticker = time.NewTicker(10 * time.Second)
		for range ticker.C {
			stat, err := mem.VirtualMemory()
			if err == nil {
				availableMemoryGB = int(stat.Available >> 30)
			}
		}
	})
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

	availableMemoryGB = int(stat.Available >> 30)
	systemTotalMemory = int(stat.Total >> 30)
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

// AvailableMemoryGB 获取当下可用内存GB数
func AvailableMemoryGB() int {
	return availableMemoryGB
}
