package caches

import (
	"github.com/cespare/xxhash"
	"sync"
)

type MemoryWriter struct {
	key            string
	expiredAt      int64
	m              map[uint64]*MemoryItem
	locker         *sync.RWMutex
	isFirstWriting bool
	headerSize     int64
	bodySize       int64
	status         int
}

func NewMemoryWriter(m map[uint64]*MemoryItem, key string, expiredAt int64, status int, locker *sync.RWMutex) *MemoryWriter {
	return &MemoryWriter{
		m:              m,
		key:            key,
		expiredAt:      expiredAt,
		locker:         locker,
		isFirstWriting: true,
		status:         status,
	}
}

// 写入数据
func (this *MemoryWriter) WriteHeader(data []byte) (n int, err error) {
	this.headerSize += int64(len(data))

	hash := this.hash(this.key)
	this.locker.Lock()
	item, ok := this.m[hash]
	if ok {
		// 第一次写先清空
		if this.isFirstWriting {
			item.HeaderValue = nil
			item.BodyValue = nil
			this.isFirstWriting = false
		}
		item.HeaderValue = append(item.HeaderValue, data...)
	} else {
		item := &MemoryItem{}
		item.HeaderValue = append([]byte{}, data...)
		item.ExpiredAt = this.expiredAt
		item.Status = this.status
		this.m[hash] = item
		this.isFirstWriting = false
	}
	this.locker.Unlock()
	return len(data), nil
}

// 写入数据
func (this *MemoryWriter) Write(data []byte) (n int, err error) {
	this.bodySize += int64(len(data))

	hash := this.hash(this.key)
	this.locker.Lock()
	item, ok := this.m[hash]
	if ok {
		// 第一次写先清空
		if this.isFirstWriting {
			item.HeaderValue = nil
			item.BodyValue = nil
			this.isFirstWriting = false
		}
		item.BodyValue = append(item.BodyValue, data...)
	} else {
		item := &MemoryItem{}
		item.BodyValue = append([]byte{}, data...)
		item.ExpiredAt = this.expiredAt
		item.Status = this.status
		this.m[hash] = item
		this.isFirstWriting = false
	}
	this.locker.Unlock()
	return len(data), nil
}

// 数据尺寸
func (this *MemoryWriter) HeaderSize() int64 {
	return this.headerSize
}

func (this *MemoryWriter) BodySize() int64 {
	return this.bodySize
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
	return xxhash.Sum64String(key)
}
