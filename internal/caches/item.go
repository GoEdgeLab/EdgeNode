package caches

import "time"

type Item struct {
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
