package caches

import "time"

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
}

func (this *Item) IsExpired() bool {
	return this.ExpiredAt < time.Now().Unix()
}

func (this *Item) TotalSize() int64 {
	return this.Size() + this.MetaSize + int64(len(this.Key)) + 64
}

func (this *Item) Size() int64 {
	return this.HeaderSize + this.BodySize
}
