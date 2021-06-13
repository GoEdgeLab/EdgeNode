package caches

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils"
)

type ItemType = int

const (
	ItemTypeFile   ItemType = 1
	ItemTypeMemory ItemType = 2
)

type Item struct {
	Type       ItemType `json:"type"`
	Key        string   `json:"key"`
	ExpiredAt  int64    `json:"expiredAt"`
	HeaderSize int64    `json:"headerSize"`
	BodySize   int64    `json:"bodySize"`
	MetaSize   int64    `json:"metaSize"`
	Host       string   `json:"host"`     // 主机名
	ServerId   int64    `json:"serverId"` // 服务ID
}

func (this *Item) IsExpired() bool {
	return this.ExpiredAt < utils.UnixTime()
}

func (this *Item) TotalSize() int64 {
	return this.Size() + this.MetaSize + int64(len(this.Key)) + 64
}

func (this *Item) Size() int64 {
	return this.HeaderSize + this.BodySize
}
