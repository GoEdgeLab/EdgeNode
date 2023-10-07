// Copyright 2023 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package fsutils

import (
	"encoding/json"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/iwind/TeaGo/Tea"
	"github.com/shirou/gopsutil/v3/load"
	"os"
	"sync/atomic"
	"time"
)

type Speed int

func (this Speed) String() string {
	switch this {
	case SpeedExtremelyFast:
		return "extremely fast"
	case SpeedFast:
		return "fast"
	case SpeedLow:
		return "low"
	case SpeedExtremelySlow:
		return "extremely slow"
	}
	return "unknown"
}

const (
	SpeedExtremelyFast Speed = 1
	SpeedFast          Speed = 2
	SpeedLow           Speed = 3
	SpeedExtremelySlow Speed = 4
)

var (
	DiskSpeed           = SpeedLow
	DiskMaxWrites int32 = 32
	DiskSpeedMB   float64
)

var IsInHighLoad = false
var IsInExtremelyHighLoad = false

const (
	highLoad1Threshold          = 20
	extremelyHighLoad1Threshold = 40
)

func init() {
	if !teaconst.IsMain {
		return
	}

	// test disk
	go func() {
		// load last result from local disk
		cacheData, cacheErr := os.ReadFile(Tea.Root + "/data/" + diskSpeedDataFile)
		if cacheErr == nil {
			var cache = &DiskSpeedCache{}
			err := json.Unmarshal(cacheData, cache)
			if err == nil && cache.SpeedMB > 0 {
				DiskSpeedMB = cache.SpeedMB
				DiskSpeed = cache.Speed
				calculateDiskMaxWrites()
			}
		}

		// initial check
		_, _, _ = CheckDiskIsFast()

		// check every one hour
		var ticker = time.NewTicker(1 * time.Hour)
		var count = 0
		for range ticker.C {
			_, _, err := CheckDiskIsFast()
			if err == nil {
				count++
				if count > 24 {
					return
				}
			}
		}
	}()

	// check high load
	go func() {
		var ticker = time.NewTicker(5 * time.Second)
		for range ticker.C {
			stat, _ := load.Avg()
			IsInExtremelyHighLoad = stat != nil && stat.Load1 > extremelyHighLoad1Threshold
			IsInHighLoad = stat != nil && stat.Load1 > highLoad1Threshold && !DiskIsFast()
		}
	}()
}

func DiskIsFast() bool {
	return DiskSpeed == SpeedExtremelyFast || DiskSpeed == SpeedFast
}

func DiskIsExtremelyFast() bool {
	// 在开发环境下返回false，以便于测试
	if Tea.IsTesting() {
		return false
	}
	return DiskSpeed == SpeedExtremelyFast
}

var countWrites int32 = 0

func WriteReady() bool {
	if IsInExtremelyHighLoad {
		return false
	}

	return atomic.LoadInt32(&countWrites) < DiskMaxWrites
}

func WriteBegin() {
	atomic.AddInt32(&countWrites, 1)
}

func WriteEnd() {
	atomic.AddInt32(&countWrites, -1)
}

func calculateDiskMaxWrites() {
	switch DiskSpeed {
	case SpeedExtremelyFast:
		DiskMaxWrites = 256
	case SpeedFast:
		DiskMaxWrites = 128
	case SpeedLow:
		DiskMaxWrites = 32
	case SpeedExtremelySlow:
		DiskMaxWrites = 16
	default:
		DiskMaxWrites = 16
	}
}
