// Copyright 2023 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package fsutils

import (
	"bytes"
	"encoding/json"
	"github.com/iwind/TeaGo/Tea"
	"math"
	"os"
	"time"
)

const diskSpeedDataFile = "disk.speed.json"

type DiskSpeedCache struct {
	Speed   Speed   `json:"speed"`
	SpeedMB float64 `json:"speedMB"`
}

// CheckDiskWritingSpeed test disk writing speed
func CheckDiskWritingSpeed() (speedMB float64, err error) {
	var tempDir = os.TempDir()
	if len(tempDir) == 0 {
		tempDir = "/tmp"
	}

	const filename = "edge-disk-writing-test.data"
	var path = tempDir + "/" + filename
	_ = os.Remove(path) // always try to delete the file

	fp, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return 0, err
	}

	var isClosed bool
	defer func() {
		if !isClosed {
			_ = fp.Close()
		}

		_ = os.Remove(path)
	}()

	var data = bytes.Repeat([]byte{'A'}, 16<<20)
	var before = time.Now()
	_, err = fp.Write(data)
	if err != nil {
		return 0, err
	}

	err = fp.Sync()
	if err != nil {
		return 0, err
	}

	err = fp.Close()
	if err != nil {
		return 0, err
	}

	var costSeconds = time.Since(before).Seconds()
	speedMB = float64(len(data)) / (1 << 20) / costSeconds
	speedMB = math.Ceil(speedMB/10) * 10

	isClosed = true

	return
}

// CheckDiskIsFast check disk is 'fast' disk to write
func CheckDiskIsFast() (speedMB float64, isFast bool, err error) {
	speedMB, err = CheckDiskWritingSpeed()
	if err != nil {
		return
	}

	isFast = speedMB > 150

	if speedMB <= DiskSpeedMB {
		return
	}

	if speedMB > 1000 {
		DiskSpeed = SpeedExtremelyFast
	} else if speedMB > 150 {
		DiskSpeed = SpeedFast
	} else if speedMB > 60 {
		DiskSpeed = SpeedLow
	} else {
		DiskSpeed = SpeedExtremelySlow
	}
	calculateDiskMaxWrites()

	DiskSpeedMB = speedMB

	// write to local file
	cacheData, jsonErr := json.Marshal(&DiskSpeedCache{
		Speed:   DiskSpeed,
		SpeedMB: DiskSpeedMB,
	})
	if jsonErr == nil {
		_ = os.WriteFile(Tea.Root+"/data/"+diskSpeedDataFile, cacheData, 0666)
	}

	return
}
