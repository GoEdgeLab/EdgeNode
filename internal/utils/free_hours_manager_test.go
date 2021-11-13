// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package utils

import (
	"testing"
	"time"
)

func TestFreeHoursManager_Update(t *testing.T) {
	var manager = NewFreeHoursManager()
	manager.Update(111)

	manager.dayTrafficMap[1] = [24]uint64{1, 1, 1, 1, 0, 0, 0, 1, 0, 1, 1, 1, 1}
	manager.dayTrafficMap[2] = [24]uint64{0, 0, 1, 0, 1, 1, 1, 0, 0}
	manager.dayTrafficMap[3] = [24]uint64{0, 1, 1, 1, 1, 0, 0, 0, 0, 1, 1, 1}
	manager.dayTrafficMap[4] = [24]uint64{0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1}
	manager.dayTrafficMap[5] = [24]uint64{0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1}
	manager.dayTrafficMap[6] = [24]uint64{0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 2, 1}
	manager.dayTrafficMap[7] = [24]uint64{}
	manager.dayTrafficMap[8] = [24]uint64{}
	manager.dayTrafficMap[9] = [24]uint64{}
	manager.dayTrafficMap[10] = [24]uint64{1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
	manager.dayTrafficMap[11] = [24]uint64{1}
	manager.dayTrafficMap[12] = [24]uint64{1}
	manager.dayTrafficMap[13] = [24]uint64{1}
	manager.dayTrafficMap[14] = [24]uint64{}
	manager.dayTrafficMap[15] = [24]uint64{}
	manager.dayTrafficMap[16] = [24]uint64{}
	manager.dayTrafficMap[25] = [24]uint64{}
	manager.dayTrafficMap[26] = [24]uint64{}
	manager.dayTrafficMap[27] = [24]uint64{}
	manager.dayTrafficMap[28] = [24]uint64{}
	manager.dayTrafficMap[29] = [24]uint64{}
	manager.dayTrafficMap[30] = [24]uint64{}
	manager.dayTrafficMap[31] = [24]uint64{}

	var before = time.Now()
	manager.Update(222)
	t.Log(manager.freeHours)
	t.Log(manager.IsFreeHour())
	t.Log(time.Since(before).Seconds()*1000, "ms")
}

func TestFreeHoursManager_SortArray(t *testing.T) {
	var manager = NewFreeHoursManager()
	t.Log(manager.sortUintArrayWeights([24]uint64{0, 1, 2, 3, 4, 5, 6, 7, 8, 109, 10, 11, 12, 130, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23}))
	t.Log(manager.sortUintArrayIndexes([24]uint64{1, 2, 3, 5, 4, 0, 100}))
}
