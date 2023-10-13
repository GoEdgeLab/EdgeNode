// Copyright 2023 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package cachehits

import (
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/TeaOSLab/EdgeNode/internal/utils/fasttime"
	"sync"
	"sync/atomic"
	"time"
)

const countSamples = 100_000

type Item struct {
	countHits   uint64
	countCached uint64
	timestamp   int64

	isGood bool
	isBad  bool
}

type Stat struct {
	goodRatio uint64
	maxItems  int

	itemMap map[string]*Item // category => *Item
	mu      *sync.RWMutex

	ticker *time.Ticker
}

func NewStat(goodRatio uint64) *Stat {
	if goodRatio == 0 {
		goodRatio = 5
	}

	var maxItems = utils.SystemMemoryGB() * 10_000
	if maxItems <= 0 {
		maxItems = 100_000
	}

	var stat = &Stat{
		goodRatio: goodRatio,
		itemMap:   map[string]*Item{},
		mu:        &sync.RWMutex{},
		ticker:    time.NewTicker(24 * time.Hour),
		maxItems:  maxItems,
	}

	goman.New(func() {
		stat.init()
	})
	return stat
}

func (this *Stat) init() {
	for range this.ticker.C {
		var currentTime = fasttime.Now().Unix()

		this.mu.RLock()
		for _, item := range this.itemMap {
			if item.timestamp < currentTime-7*24*86400 {
				// reset
				item.countHits = 0
				item.countCached = 1
				item.timestamp = currentTime
				item.isGood = false
				item.isBad = false
			}
		}
		this.mu.RUnlock()
	}
}

func (this *Stat) IncreaseCached(category string) {
	this.mu.RLock()
	var item = this.itemMap[category]
	if item != nil {
		if item.isGood || item.isBad {
			this.mu.RUnlock()
			return
		}

		atomic.AddUint64(&item.countCached, 1)
		this.mu.RUnlock()
		return
	}
	this.mu.RUnlock()

	this.mu.Lock()

	if len(this.itemMap) > this.maxItems {
		// remove one randomly
		for k := range this.itemMap {
			delete(this.itemMap, k)
			break
		}
	}

	this.itemMap[category] = &Item{
		countHits:   0,
		countCached: 1,
		timestamp:   fasttime.Now().Unix(),
	}
	this.mu.Unlock()
}

func (this *Stat) IncreaseHit(category string) {
	this.mu.RLock()
	defer this.mu.RUnlock()

	var item = this.itemMap[category]
	if item != nil {
		if item.isGood || item.isBad {
			return
		}

		atomic.AddUint64(&item.countHits, 1)
		return
	}
}

func (this *Stat) IsGood(category string) bool {
	this.mu.RLock()
	defer func() {
		this.mu.RUnlock()
	}()

	var item = this.itemMap[category]
	if item != nil {
		if item.isBad {
			return false
		}
		if item.isGood {
			return true
		}

		if item.countCached > countSamples && item.timestamp < fasttime.Now().Unix()-600 /** 10 minutes ago **/ {
			var isGood = item.countHits*100/item.countCached >= this.goodRatio
			if isGood {
				item.isGood = true
			} else {
				item.isBad = true
			}

			return isGood
		}
	}

	return true
}

func (this *Stat) Len() int {
	this.mu.RLock()
	defer this.mu.RUnlock()

	return len(this.itemMap)
}
