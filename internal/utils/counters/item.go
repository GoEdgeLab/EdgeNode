// Copyright 2023 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package counters

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils/fasttime"
)

type Item struct {
	lifeSeconds int64

	spanSeconds int64
	spans       []*Span

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
	var spans = []*Span{}
	for i := 0; i < countSpans; i++ {
		spans = append(spans, NewSpan())
	}

	return &Item{
		lifeSeconds:    int64(lifeSeconds),
		spanSeconds:    int64(spanSeconds),
		spans:          spans,
		lastUpdateTime: fasttime.Now().Unix(),
	}
}

func (this *Item) Increase() uint64 {
	var currentTime = fasttime.Now().Unix()
	var spanIndex = int(currentTime % this.lifeSeconds / this.spanSeconds)
	var span = this.spans[spanIndex]
	var roundTime = currentTime / this.spanSeconds * this.spanSeconds

	this.lastUpdateTime = currentTime

	if span.Timestamp < roundTime {
		span.Timestamp = roundTime // update time
		span.Count = 0             // reset count
	}
	span.Count++

	return this.Sum()
}

func (this *Item) Sum() uint64 {
	var result uint64 = 0
	var currentTimestamp = fasttime.Now().Unix()
	for _, span := range this.spans {
		if span.Timestamp >= currentTimestamp-this.lifeSeconds {
			result += span.Count
		}
	}
	return result
}

func (this *Item) Reset() {
	for _, span := range this.spans {
		span.Count = 0
		span.Timestamp = 0
	}
}

func (this *Item) IsExpired(currentTime int64) bool {
	return this.lastUpdateTime < currentTime-this.lifeSeconds-this.spanSeconds
}
