// Copyright 2023 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package fsutils

import (
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"sync/atomic"
	"time"
)

var (
	DiskIsFast  bool
	DiskSpeedMB float64
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

var countWrites int32 = 0

const MaxWrites = 32
const MaxFastWrites = 128

func WriteReady() bool {
	var count = atomic.LoadInt32(&countWrites)
	if DiskIsFast {
		return count < MaxFastWrites
	}
	return count <= MaxWrites
}

func WriteBegin() {
	atomic.AddInt32(&countWrites, 1)
}

func WriteEnd() {
	atomic.AddInt32(&countWrites, -1)
}
