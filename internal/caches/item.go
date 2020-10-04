package caches

import "time"

type Item struct {
	Key       string
	ExpiredAt int64
	ValueSize int64
	Size      int64
}

func (this *Item) IsExpired() bool {
	return this.ExpiredAt < time.Now().Unix()
}
