package caches

import (
	"github.com/dchest/siphash"
	"sync"
)

type MemoryWriter struct {
	key            string
	expiredAt      int64
	m              map[uint64]*MemoryItem
	locker         *sync.RWMutex
	isFirstWriting bool
	size           int64
}

func NewMemoryWriter(m map[uint64]*MemoryItem, key string, expiredAt int64, locker *sync.RWMutex) *MemoryWriter {
	return &MemoryWriter{
		m:              m,
		key:            key,
		expiredAt:      expiredAt,
		locker:         locker,
		isFirstWriting: true,
	}
}

// 写入数据
func (this *MemoryWriter) Write(data []byte) (n int, err error) {
	this.size += int64(len(data))

	hash := this.hash(this.key)
	this.locker.Lock()
	item, ok := this.m[hash]
	if ok {
		// 第一次写先清空
		if this.isFirstWriting {
			item.Value = nil
			this.isFirstWriting = false
		}
		item.Value = append(item.Value, data...)
	} else {
		item := &MemoryItem{}
		item.Value = append([]byte{}, data...)
		item.ExpiredAt = this.expiredAt
		this.m[hash] = item
		this.isFirstWriting = false
	}
	this.locker.Unlock()
	return len(data), nil
}

// 数据尺寸
func (this *MemoryWriter) Size() int64 {
	return this.size
}

// 关闭
func (this *MemoryWriter) Close() error {
	return nil
}

// 丢弃
func (this *MemoryWriter) Discard() error {
	hash := this.hash(this.key)
	this.locker.Lock()
	delete(this.m, hash)
	this.locker.Unlock()
	return nil
}

// Key
func (this *MemoryWriter) Key() string {
	return this.key
}

// 过期时间
func (this *MemoryWriter) ExpiredAt() int64 {
	return this.expiredAt
}

// 计算Key Hash
func (this *MemoryWriter) hash(key string) uint64 {
	return siphash.Hash(0, 0, []byte(key))
}
