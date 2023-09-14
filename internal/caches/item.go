package caches

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils/fasttime"
	"strings"
)

type ItemType = int

const (
	ItemTypeFile   ItemType = 1
	ItemTypeMemory ItemType = 2
)

// 计算当前周
// 不要用YW，因为需要计算两周是否临近
func currentWeek() int32 {
	return int32(fasttime.Now().Unix() / 86400)
}

type Item struct {
	Type       ItemType `json:"type"`
	Key        string   `json:"key"`
	ExpiredAt  int64    `json:"expiredAt"`
	StaleAt    int64    `json:"staleAt"`
	HeaderSize int64    `json:"headerSize"`
	BodySize   int64    `json:"bodySize"`
	MetaSize   int64    `json:"metaSize"`
	Host       string   `json:"host"`     // 主机名
	ServerId   int64    `json:"serverId"` // 服务ID
	Week       int32    `json:"week"`
}

func (this *Item) IsExpired() bool {
	return this.ExpiredAt < fasttime.Now().Unix()
}

func (this *Item) TotalSize() int64 {
	return this.Size() + this.MetaSize + int64(len(this.Key)) + int64(len(this.Host))
}

func (this *Item) Size() int64 {
	return this.HeaderSize + this.BodySize
}

func (this *Item) RequestURI() string {
	var schemeIndex = strings.Index(this.Key, "://")
	if schemeIndex <= 0 {
		return ""
	}

	var firstSlashIndex = strings.Index(this.Key[schemeIndex+3:], "/")
	if firstSlashIndex <= 0 {
		return ""
	}

	return this.Key[schemeIndex+3+firstSlashIndex:]
}
