// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package idles

import (
	"encoding/json"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	fsutils "github.com/TeaOSLab/EdgeNode/internal/utils/fs"
	"github.com/iwind/TeaGo/Tea"
	"github.com/shirou/gopsutil/v3/load"
	"math"
	"os"
	"sort"
	"time"
)

const maxSamples = 7
const cacheFile = "idle_hours.cache"

var hourlyLoadMap = map[int]*HourlyLoad{}
var sharedMinLoadHours []int

type HourlyLoad struct {
	Hour   int       `json:"hour"`
	Avg    float64   `json:"avg"`
	Values []float64 `json:"values"`
}

func init() {
	if !teaconst.IsMain {
		return
	}

	// recover from cache
	{
		data, err := os.ReadFile(Tea.Root + "/data/" + cacheFile)
		if err == nil {
			err = json.Unmarshal(data, &hourlyLoadMap)
			if err == nil {
				calculateMinLoadHours()
			}
		}
	}

	goman.New(func() {
		var ticker = time.NewTicker(1 * time.Hour)
		for range ticker.C {
			CheckHourlyLoad(time.Now().Hour())
		}
	})
}

func CheckHourlyLoad(hour int) {
	avgLoad, err := load.Avg()
	if err != nil {
		return
	}

	hourlyLoad, ok := hourlyLoadMap[hour]
	if !ok {
		hourlyLoad = &HourlyLoad{
			Hour: hour,
		}
		hourlyLoadMap[hour] = hourlyLoad
	}

	if len(hourlyLoad.Values) >= maxSamples {
		hourlyLoad.Values = hourlyLoad.Values[:maxSamples-1]
	}
	hourlyLoad.Values = append(hourlyLoad.Values, avgLoad.Load15)

	var sum float64
	for _, v := range hourlyLoad.Values {
		sum += v
	}
	hourlyLoad.Avg = math.Ceil(sum/float64(len(hourlyLoad.Values))*10) / 10 // fix precision

	calculateMinLoadHours()
}

func Run(f func()) {
	defer f()

	var minLoadHours = sharedMinLoadHours // copy

	if len(minLoadHours) == 0 {
		fsutils.WaitLoad(15, 8, time.Hour)
		return
	}

	var hour = time.Now().Hour()
	var minLoadHour = -1
	for _, v := range minLoadHours {
		if v == hour {
			minLoadHour = v
			break
		}
		if v > hour {
			minLoadHour = v
			break
		}
	}
	if minLoadHour < 0 {
		minLoadHour = minLoadHours[0]
	}

	if minLoadHour == hour {
		fsutils.WaitLoad(15, 10, time.Minute)
		return
	}

	if minLoadHour < hour {
		time.Sleep(time.Duration(24-hour+minLoadHour) * time.Hour)
	} else {
		time.Sleep(time.Duration(minLoadHour-hour) * time.Hour)
	}
	fsutils.WaitLoad(15, 10, time.Minute)
}

func RunTicker(ticker *time.Ticker, f func()) {
	for range ticker.C {
		Run(f)
	}
}

func calculateMinLoadHours() {
	var allLoads = []*HourlyLoad{}
	for _, v := range hourlyLoadMap {
		allLoads = append(allLoads, v)
	}

	sort.Slice(allLoads, func(i, j int) bool {
		return allLoads[i].Avg < allLoads[j].Avg
	})

	var minAvgLoad = allLoads[0].Avg
	var newMinLoadHours []int
	for _, v := range allLoads {
		if v.Avg == minAvgLoad {
			newMinLoadHours = append(newMinLoadHours, v.Hour)
		}
	}
	sort.Ints(newMinLoadHours)
	sharedMinLoadHours = newMinLoadHours

	// write to cache
	hourlyLoadMapJSON, err := json.Marshal(hourlyLoadMap)
	if err == nil {
		_ = os.WriteFile(Tea.Root+"/data/"+cacheFile, hourlyLoadMapJSON, 0666)
	}
}

func TestMinLoadHours() []int {
	return sharedMinLoadHours
}

func TestSetMinLoadHours(minLoadHours []int) {
	sharedMinLoadHours = minLoadHours
}

func TestHourlyLoadMap() map[int]*HourlyLoad {
	return hourlyLoadMap
}
