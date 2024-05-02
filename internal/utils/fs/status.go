// Copyright 2023 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package fsutils

import (
	"encoding/json"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/iwind/TeaGo/Tea"
	"github.com/shirou/gopsutil/v3/load"
	"os"
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
	DiskSpeed   = SpeedLow
	DiskSpeedMB float64
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
	goman.New(func() {
		// load last result from local disk
		var countTests int
		cacheData, cacheErr := os.ReadFile(Tea.Root + "/data/" + diskSpeedDataFile)
		if cacheErr == nil {
			var cache = &DiskSpeedCache{}
			err := json.Unmarshal(cacheData, cache)
			if err == nil && cache.SpeedMB > 0 {
				DiskSpeedMB = cache.SpeedMB
				DiskSpeed = cache.Speed
				countTests = cache.CountTests
			}
		}

		if countTests < 12 {
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
		}
	})

	// check high load
	goman.New(func() {
		var ticker = time.NewTicker(5 * time.Second)
		for range ticker.C {
			stat, _ := load.Avg()
			IsInExtremelyHighLoad = stat != nil && stat.Load1 > extremelyHighLoad1Threshold
			IsInHighLoad = stat != nil && stat.Load1 > highLoad1Threshold && !DiskIsFast()
		}
	})
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

// WaitLoad wait system load to downgrade
func WaitLoad(maxLoad float64, maxLoops int, delay time.Duration) {
	for i := 0; i < maxLoops; i++ {
		stat, err := load.Avg()
		if err == nil {
			if stat.Load1 > maxLoad {
				time.Sleep(delay)
			} else {
				return
			}
		}
	}
}
