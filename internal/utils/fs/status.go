// Copyright 2023 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package fsutils

import (
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
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

func init() {
	if !teaconst.IsMain {
		return
	}

	// test disk
	go func() {
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
}

func DiskIsFast() bool {
	return DiskSpeed == SpeedExtremelyFast || DiskSpeed == SpeedFast
}

var countWrites int32 = 0

func WriteReady() bool {
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
