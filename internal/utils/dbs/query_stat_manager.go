// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package dbs

import (
	"fmt"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/iwind/TeaGo/logs"
	"sort"
	"strings"
	"sync"
	"time"
)

func init() {
	if !teaconst.IsMain {
		return
	}

	var ticker = time.NewTicker(5 * time.Second)

	events.On(events.EventLoaded, func() {
		if teaconst.EnableDBStat {
			goman.New(func() {
				for range ticker.C {
					var stats = []string{}
					for _, stat := range SharedQueryStatManager.TopN(10) {
						var avg = stat.CostTotal / float64(stat.Calls)
						var query = stat.Query
						if len(query) > 128 {
							query = query[:128]
						}
						stats = append(stats, fmt.Sprintf("%.2fms/%.2fms/%.2fms - %d - %s", stat.CostMin*1000, stat.CostMax*1000, avg*1000, stat.Calls, query))
					}
					logs.Println("\n========== DB STATS ==========\n" + strings.Join(stats, "\n") + "\n=============================")
				}
			})
		}
	})
}

var SharedQueryStatManager = NewQueryStatManager()

type QueryStatManager struct {
	statsMap map[string]*QueryStat // query => *QueryStat
	locker   sync.Mutex
}

func NewQueryStatManager() *QueryStatManager {
	return &QueryStatManager{
		statsMap: map[string]*QueryStat{},
	}
}

func (this *QueryStatManager) AddQuery(query string) *QueryLabel {
	return NewQueryLabel(this, query)
}

func (this *QueryStatManager) AddCost(query string, cost float64) {
	this.locker.Lock()
	defer this.locker.Unlock()

	stat, ok := this.statsMap[query]
	if !ok {
		stat = NewQueryStat(query)
		this.statsMap[query] = stat
	}
	stat.AddCost(cost)
}

func (this *QueryStatManager) TopN(n int) []*QueryStat {
	this.locker.Lock()
	defer this.locker.Unlock()

	var stats = []*QueryStat{}
	for _, stat := range this.statsMap {
		stats = append(stats, stat)
	}
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].CostMax > stats[j].CostMax
	})

	if len(stats) > n {
		return stats[:n]
	}
	return stats
}
