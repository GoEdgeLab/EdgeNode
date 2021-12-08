// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package utils

import (
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

var SharedFreeHoursManager = NewFreeHoursManager()

func init() {
	events.On(events.EventLoaded, func() {
		goman.New(func() {
			SharedFreeHoursManager.Start()
		})
	})
}

// FreeHoursManager 计算节点空闲时间
// 以便于我们在空闲时间执行高强度的任务，如果清理缓存等
type FreeHoursManager struct {
	dayTrafficMap map[int][24]uint64 // day => [ traffic bytes ]
	lastBytes     uint64

	freeHours []int
	count     int

	locker sync.Mutex
}

func NewFreeHoursManager() *FreeHoursManager {
	return &FreeHoursManager{dayTrafficMap: map[int][24]uint64{}, count: 3}
}

func (this *FreeHoursManager) Start() {
	var ticker = time.NewTicker(30 * time.Minute)
	for range ticker.C {
		this.Update(atomic.LoadUint64(&teaconst.InTrafficBytes))
	}
}

func (this *FreeHoursManager) Update(bytes uint64) {
	if this.count <= 0 {
		this.count = 3
	}

	if this.lastBytes == 0 {
		this.lastBytes = bytes
	} else {
		// 记录流量
		var deltaBytes = bytes - this.lastBytes
		var now = time.Now()
		var day = now.Day()
		var hour = now.Hour()
		traffic, ok := this.dayTrafficMap[day]
		if ok {
			traffic[hour] += deltaBytes
		} else {
			var traffic = [24]uint64{}
			traffic[hour] += deltaBytes
			this.dayTrafficMap[day] = traffic
		}

		this.lastBytes = bytes

		// 计算空闲时间
		var result = [24]uint64{}
		var hasData = false
		for trafficDay, trafficArray := range this.dayTrafficMap {
			// 当天的不算
			if trafficDay == day {
				continue
			}

			// 查看最近5天的
			if (day > trafficDay && day-trafficDay <= 5) || (day < trafficDay && trafficDay-day >= 26) {
				var weights = this.sortUintArrayWeights(trafficArray)
				for k, v := range weights {
					result[k] += v
				}
				hasData = true
			}
		}
		if hasData {
			var freeHours = this.sortUintArrayIndexes(result)
			this.locker.Lock()
			this.freeHours = freeHours[:this.count] // 取前N个小时作为空闲时间
			this.locker.Unlock()
		}
	}
}

func (this *FreeHoursManager) IsFreeHour() bool {
	this.locker.Lock()
	defer this.locker.Unlock()

	if len(this.freeHours) == 0 {
		return false
	}

	var hour = time.Now().Hour()
	for _, h := range this.freeHours {
		if h == hour {
			return true
		}
	}
	return false
}

// 对数组进行排序，并返回权重
func (this *FreeHoursManager) sortUintArrayWeights(arr [24]uint64) [24]uint64 {
	var l = []map[string]interface{}{}
	for k, v := range arr {
		l = append(l, map[string]interface{}{
			"k": k,
			"v": v,
		})
	}
	sort.Slice(l, func(i, j int) bool {
		var m1 = l[i]
		var v1 = m1["v"].(uint64)

		var m2 = l[j]
		var v2 = m2["v"].(uint64)

		return v1 < v2
	})

	var result = [24]uint64{}
	for k, v := range l {
		if k < this.count {
			k = 0
		} else {
			k = 1
		}
		result[v["k"].(int)] = v["v"].(uint64)
	}

	return result
}

// 对数组进行排序，并返回索引
func (this *FreeHoursManager) sortUintArrayIndexes(arr [24]uint64) [24]int {
	var l = []map[string]interface{}{}
	for k, v := range arr {
		l = append(l, map[string]interface{}{
			"k": k,
			"v": v,
		})
	}
	sort.Slice(l, func(i, j int) bool {
		var m1 = l[i]
		var v1 = m1["v"].(uint64)

		var m2 = l[j]
		var v2 = m2["v"].(uint64)

		return v1 < v2
	})

	var result = [24]int{}
	var i = 0
	for _, v := range l {
		result[i] = v["k"].(int)
		i++
	}

	return result
}
