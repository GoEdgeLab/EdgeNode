package caches

import (
	"github.com/cespare/xxhash"
	"sync"
)

type MemoryWriter struct {
	key        string
	expiredAt  int64
	m          map[uint64]*MemoryItem
	locker     *sync.RWMutex
	headerSize int64
	bodySize   int64
	status     int

	hash uint64
	item *MemoryItem
}

func NewMemoryWriter(m map[uint64]*MemoryItem, key string, expiredAt int64, status int, locker *sync.RWMutex) *MemoryWriter {
	w := &MemoryWriter{
		m:         m,
		key:       key,
		expiredAt: expiredAt,
		locker:    locker,
		item: &MemoryItem{
			ExpiredAt: expiredAt,
			Status:    status,
		},
		status: status,
	}
	w.hash = w.calculateHash(key)

	return w
}

// 写入数据
func (this *MemoryWriter) WriteHeader(data []byte) (n int, err error) {
	this.headerSize += int64(len(data))

	this.locker.Lock()
	this.item.HeaderValue = append(this.item.HeaderValue, data...)
	this.locker.Unlock()
	return len(data), nil
}

// 写入数据
func (this *MemoryWriter) Write(data []byte) (n int, err error) {
	this.bodySize += int64(len(data))

	this.locker.Lock()
	this.item.BodyValue = append(this.item.BodyValue, data...)
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
	if this.item == nil {
		return nil
	}

	this.locker.Lock()
	this.item.IsDone = true
	this.m[this.hash] = this.item
	this.locker.Unlock()

	return nil
}

// 丢弃
func (this *MemoryWriter) Discard() error {
	this.locker.Lock()
	delete(this.m, this.hash)
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

// 内容类型
func (this *MemoryWriter) ItemType() ItemType {
	return ItemTypeMemory
}

// 计算Key Hash
func (this *MemoryWriter) calculateHash(key string) uint64 {
	return xxhash.Sum64String(key)
}
