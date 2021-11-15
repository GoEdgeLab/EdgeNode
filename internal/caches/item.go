package caches

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"time"
)

type ItemType = int

const (
	ItemTypeFile   ItemType = 1
	ItemTypeMemory ItemType = 2
)

// 计算当前周
// 不要用YW，因为需要计算两周是否临近
func currentWeek() int32 {
	return int32(time.Now().Unix() / 86400)
}

type Item struct {
	Type       ItemType `json:"type"`
	Key        string   `json:"key"`
	ExpiredAt  int64    `json:"expiredAt"`
	HeaderSize int64    `json:"headerSize"`
	BodySize   int64    `json:"bodySize"`
	MetaSize   int64    `json:"metaSize"`
	Host       string   `json:"host"`     // 主机名
	ServerId   int64    `json:"serverId"` // 服务ID

	Week1Hits int64 `json:"week1Hits"`
	Week2Hits int64 `json:"week2Hits"`
	Week      int32 `json:"week"`
}

func (this *Item) IsExpired() bool {
	return this.ExpiredAt < utils.UnixTime()
}

func (this *Item) TotalSize() int64 {
	return this.Size() + this.MetaSize + int64(len(this.Key)) + int64(len(this.Host))
}

func (this *Item) Size() int64 {
	return this.HeaderSize + this.BodySize
}

func (this *Item) IncreaseHit(week int32) {
	if this.Week == week {
		this.Week2Hits++
	} else {
		if week-this.Week == 1 {
			this.Week1Hits = this.Week2Hits
		} else {
			this.Week1Hits = 0
		}
		this.Week2Hits = 1
		this.Week = week
	}
}
