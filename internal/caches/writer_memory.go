package caches

import (
	"github.com/cespare/xxhash"
	"sync"
	"time"
)

type MemoryWriter struct {
	key        string
	expiredAt  int64
	m          map[uint64]*MemoryItem
	locker     *sync.RWMutex
	headerSize int64
	bodySize   int64
	status     int

	hash    uint64
	item    *MemoryItem
	endFunc func()
}

func NewMemoryWriter(m map[uint64]*MemoryItem, key string, expiredAt int64, status int, locker *sync.RWMutex, endFunc func()) *MemoryWriter {
	w := &MemoryWriter{
		m:         m,
		key:       key,
		expiredAt: expiredAt,
		locker:    locker,
		item: &MemoryItem{
			ExpiredAt:  expiredAt,
			ModifiedAt: time.Now().Unix(),
			Status:     status,
		},
		status:  status,
		endFunc: endFunc,
	}
	w.hash = w.calculateHash(key)

	return w
}

// WriteHeader 写入数据
func (this *MemoryWriter) WriteHeader(data []byte) (n int, err error) {
	this.headerSize += int64(len(data))
	this.item.HeaderValue = append(this.item.HeaderValue, data...)
	return len(data), nil
}

// Write 写入数据
func (this *MemoryWriter) Write(data []byte) (n int, err error) {
	this.bodySize += int64(len(data))
	this.item.BodyValue = append(this.item.BodyValue, data...)
	return len(data), nil
}

// HeaderSize 数据尺寸
func (this *MemoryWriter) HeaderSize() int64 {
	return this.headerSize
}

// BodySize 主体内容尺寸
func (this *MemoryWriter) BodySize() int64 {
	return this.bodySize
}

// Close 关闭
func (this *MemoryWriter) Close() error {
	// 需要在Locker之外
	defer this.endFunc()

	if this.item == nil {
		return nil
	}

	this.locker.Lock()
	this.item.IsDone = true
	this.m[this.hash] = this.item
	this.locker.Unlock()

	return nil
}

// Discard 丢弃
func (this *MemoryWriter) Discard() error {
	// 需要在Locker之外
	defer this.endFunc()

	this.locker.Lock()
	delete(this.m, this.hash)
	this.locker.Unlock()
	return nil
}

// Key 获取Key
func (this *MemoryWriter) Key() string {
	return this.key
}

// ExpiredAt 过期时间
func (this *MemoryWriter) ExpiredAt() int64 {
	return this.expiredAt
}

// ItemType 内容类型
func (this *MemoryWriter) ItemType() ItemType {
	return ItemTypeMemory
}

// 计算Key Hash
func (this *MemoryWriter) calculateHash(key string) uint64 {
	return xxhash.Sum64String(key)
}
