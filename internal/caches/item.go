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
	Type       ItemType `json:"-"`
	Key        string   `json:"1,omitempty"`
	ExpiresAt  int64    `json:"2,omitempty"`
	StaleAt    int64    `json:"3,omitempty"`
	HeaderSize int64    `json:"-"`
	BodySize   int64    `json:"4,omitempty"`
	MetaSize   int64    `json:"-"`
	Host       string   `json:"-"`           // 主机名
	ServerId   int64    `json:"5,omitempty"` // 服务ID
	Week       int32    `json:"-"`
	CreatedAt  int64    `json:"6,omitempty"`
}

func (this *Item) IsExpired() bool {
	return this.ExpiresAt < fasttime.Now().Unix()
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
