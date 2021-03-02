package caches

import "time"

type ItemType = int

const (
	ItemTypeFile   ItemType = 1
	ItemTypeMemory ItemType = 2
)

type Item struct {
	Type       ItemType
	Key        string
	ExpiredAt  int64
	HeaderSize int64
	BodySize   int64
	MetaSize   int64
}

func (this *Item) IsExpired() bool {
	return this.ExpiredAt < time.Now().Unix()
}

func (this *Item) TotalSize() int64 {
	return this.Size() + this.MetaSize + int64(len(this.Key))
}

func (this *Item) Size() int64 {
	return this.HeaderSize + this.BodySize
}
