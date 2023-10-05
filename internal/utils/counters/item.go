// Copyright 2023 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package counters

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils/fasttime"
)

type Item struct {
	lifeSeconds int64

	spanSeconds int64
	spans       []uint64

	lastUpdateTime int64
}

func NewItem(lifeSeconds int) *Item {
	if lifeSeconds <= 0 {
		lifeSeconds = 60
	}
	var spanSeconds = lifeSeconds / 10
	if spanSeconds < 1 {
		spanSeconds = 1
	}
	var countSpans = lifeSeconds/spanSeconds + 1 /** prevent index out of bounds **/

	return &Item{
		lifeSeconds:    int64(lifeSeconds),
		spanSeconds:    int64(spanSeconds),
		spans:          make([]uint64, countSpans),
		lastUpdateTime: fasttime.Now().Unix(),
	}
}

func (this *Item) Increase() uint64 {
	var currentTime = fasttime.Now().Unix()
	var currentSpanIndex = this.calculateSpanIndex(currentTime)

	// return quickly
	if this.lastUpdateTime == currentTime {
		this.spans[currentSpanIndex]++
		return this.Sum()
	}

	if this.lastUpdateTime > 0 {
		if currentTime-this.lastUpdateTime > this.lifeSeconds {
			for index := range this.spans {
				this.spans[index] = 0
			}
		} else {
			var lastSpanIndex = this.calculateSpanIndex(this.lastUpdateTime)
			var countSpans = len(this.spans)

			// reset values between LAST and CURRENT
			for index := lastSpanIndex + 1; ; index++ {
				var realIndex = index % countSpans
				if realIndex <= currentSpanIndex {
					this.spans[realIndex] = 0
				}
				if realIndex == currentSpanIndex {
					break
				}
			}
		}
	}

	this.spans[currentSpanIndex]++
	this.lastUpdateTime = currentTime

	return this.Sum()
}

func (this *Item) Sum() (result uint64) {
	if this.lastUpdateTime == 0 {
		return 0
	}

	var currentTime = fasttime.Now().Unix()
	var currentSpanIndex = this.calculateSpanIndex(currentTime)

	if currentTime-this.lastUpdateTime > this.lifeSeconds {
		return 0
	} else {
		var lastSpanIndex = this.calculateSpanIndex(this.lastUpdateTime)
		var countSpans = len(this.spans)
		for index := 0; index < countSpans; index++ {
			if (currentSpanIndex >= lastSpanIndex && (index <= lastSpanIndex || index >= currentSpanIndex /** a >=b **/)) ||
				(currentSpanIndex < lastSpanIndex && index >= currentSpanIndex && index <= lastSpanIndex /** a < b **/) {
				result += this.spans[index]
			}
		}
	}

	return result
}

func (this *Item) Reset() {
	for index := range this.spans {
		this.spans[index] = 0
	}
}

func (this *Item) IsExpired(currentTime int64) bool {
	return this.lastUpdateTime < currentTime-this.lifeSeconds-this.spanSeconds
}

func (this *Item) calculateSpanIndex(timestamp int64) int {
	return int(timestamp % this.lifeSeconds / this.spanSeconds)
}
