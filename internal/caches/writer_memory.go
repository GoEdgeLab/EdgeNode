package caches

import (
	"errors"
	"github.com/TeaOSLab/EdgeNode/internal/utils/fasttime"
	"github.com/cespare/xxhash"
	"github.com/iwind/TeaGo/types"
	"sync"
	"sync/atomic"
)

type MemoryWriter struct {
	storage *MemoryStorage

	key        string
	expiredAt  int64
	headerSize int64
	bodySize   int64
	status     int
	isDirty    bool

	expectedBodySize int64
	maxSize          int64

	hash    uint64
	item    *MemoryItem
	endFunc func(valueItem *MemoryItem)
	once    sync.Once
}

func NewMemoryWriter(memoryStorage *MemoryStorage, key string, expiredAt int64, status int, isDirty bool, expectedBodySize int64, maxSize int64, endFunc func(valueItem *MemoryItem)) *MemoryWriter {
	var valueItem = &MemoryItem{
		ExpiresAt:  expiredAt,
		ModifiedAt: fasttime.Now().Unix(),
		Status:     status,
	}

	if expectedBodySize > 0 {
		valueItem.BodyValue = make([]byte, 0, expectedBodySize)
	}

	var w = &MemoryWriter{
		storage:          memoryStorage,
		key:              key,
		expiredAt:        expiredAt,
		item:             valueItem,
		status:           status,
		isDirty:          isDirty,
		expectedBodySize: expectedBodySize,
		maxSize:          maxSize,
		endFunc:          endFunc,
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
	var l = len(data)
	if l == 0 {
		return
	}

	if this.item.IsPrepared {
		if this.item.WriteOffset+int64(l) > this.expectedBodySize {
			err = ErrWritingUnavailable
			return
		}
		copy(this.item.BodyValue[this.item.WriteOffset:], data)
		this.item.WriteOffset += int64(l)
	} else {
		this.item.BodyValue = append(this.item.BodyValue, data...)
	}

	this.bodySize += int64(l)

	// 检查尺寸
	if this.maxSize > 0 && this.bodySize > this.maxSize {
		err = ErrEntityTooLarge
		this.storage.IgnoreKey(this.key, this.maxSize)
		return l, err
	}

	return l, nil
}

// WriteAt 在指定位置写入数据
func (this *MemoryWriter) WriteAt(offset int64, b []byte) error {
	_ = b
	_ = offset
	return errors.New("not supported")
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
	defer this.once.Do(func() {
		this.endFunc(this.item)
	})

	if this.item == nil {
		return nil
	}

	// check content length
	if this.expectedBodySize > 0 && this.bodySize != this.expectedBodySize {
		this.storage.locker.Lock()
		delete(this.storage.valuesMap, this.hash)
		this.storage.locker.Unlock()
		return ErrUnexpectedContentLength
	}

	this.storage.locker.Lock()
	this.item.IsDone = true
	var err error
	if this.isDirty {
		if this.storage.parentStorage != nil {
			this.storage.valuesMap[this.hash] = this.item

			select {
			case this.storage.dirtyChan <- types.String(this.bodySize) + "@" + this.key:
				atomic.AddInt64(&this.storage.totalDirtySize, this.bodySize)
			default:
				// remove from values map
				delete(this.storage.valuesMap, this.hash)

				err = ErrWritingQueueFull
			}
		} else {
			this.storage.valuesMap[this.hash] = this.item
		}
	} else {
		this.storage.valuesMap[this.hash] = this.item
	}

	this.storage.locker.Unlock()

	return err
}

// Discard 丢弃
func (this *MemoryWriter) Discard() error {
	// 需要在Locker之外
	defer this.once.Do(func() {
		this.endFunc(this.item)
	})

	this.storage.locker.Lock()
	delete(this.storage.valuesMap, this.hash)
	this.storage.locker.Unlock()
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
