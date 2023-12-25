// Copyright 2023 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package counters

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/TeaOSLab/EdgeNode/internal/utils/fasttime"
	syncutils "github.com/TeaOSLab/EdgeNode/internal/utils/sync"
	"github.com/cespare/xxhash"
	"sync"
	"time"
)

const maxItemsPerGroup = 50_000

var SharedCounter = NewCounter[uint32]().WithGC()

type SupportedUIntType interface {
	uint32 | uint64
}

type Counter[T SupportedUIntType] struct {
	countMaps uint64
	locker    *syncutils.RWMutex
	itemMaps  []map[uint64]Item[T]

	gcTicker *time.Ticker
	gcIndex  int
	gcLocker sync.Mutex
}

// NewCounter create new counter
func NewCounter[T SupportedUIntType]() *Counter[T] {
	var count = utils.SystemMemoryGB() * 8
	if count < 8 {
		count = 8
	}

	var itemMaps = []map[uint64]Item[T]{}
	for i := 0; i < count; i++ {
		itemMaps = append(itemMaps, map[uint64]Item[T]{})
	}

	var counter = &Counter[T]{
		countMaps: uint64(count),
		locker:    syncutils.NewRWMutex(count),
		itemMaps:  itemMaps,
	}

	return counter
}

// WithGC start the counter with gc automatically
func (this *Counter[T]) WithGC() *Counter[T] {
	if this.gcTicker != nil {
		return this
	}
	this.gcTicker = time.NewTicker(1 * time.Second)
	go func() {
		for range this.gcTicker.C {
			this.GC()
		}
	}()

	return this
}

// Increase key
func (this *Counter[T]) Increase(key uint64, lifeSeconds int) T {
	var index = int(key % this.countMaps)
	this.locker.RLock(index)
	var item = this.itemMaps[index][key] // item MUST NOT be pointer
	this.locker.RUnlock(index)
	if !item.IsOk() {
		// no need to care about duplication
		// always insert new item even when itemMap is full
		item = NewItem[T](lifeSeconds)
		var result = item.Increase()
		this.locker.Lock(index)
		this.itemMaps[index][key] = item
		this.locker.Unlock(index)
		return result
	}

	this.locker.Lock(index)
	var result = item.Increase()
	this.itemMaps[index][key] = item // overwrite
	this.locker.Unlock(index)
	return result
}

// IncreaseKey increase string key
func (this *Counter[T]) IncreaseKey(key string, lifeSeconds int) T {
	return this.Increase(this.hash(key), lifeSeconds)
}

// Get value of key
func (this *Counter[T]) Get(key uint64) T {
	var index = int(key % this.countMaps)
	this.locker.RLock(index)
	defer this.locker.RUnlock(index)
	var item = this.itemMaps[index][key]
	if item.IsOk() {
		return item.Sum()
	}
	return 0
}

// GetKey get value of string key
func (this *Counter[T]) GetKey(key string) T {
	return this.Get(this.hash(key))
}

// Reset key
func (this *Counter[T]) Reset(key uint64) {
	var index = int(key % this.countMaps)
	this.locker.RLock(index)
	var item = this.itemMaps[index][key]
	this.locker.RUnlock(index)

	if item.IsOk() {
		this.locker.Lock(index)
		delete(this.itemMaps[index], key)
		this.locker.Unlock(index)
	}
}

// ResetKey string key
func (this *Counter[T]) ResetKey(key string) {
	this.Reset(this.hash(key))
}

// TotalItems get items count
func (this *Counter[T]) TotalItems() int {
	var total = 0

	for i := 0; i < int(this.countMaps); i++ {
		this.locker.RLock(i)
		total += len(this.itemMaps[i])
		this.locker.RUnlock(i)
	}

	return total
}

// GC garbage expired items
func (this *Counter[T]) GC() {
	this.gcLocker.Lock()
	var gcIndex = this.gcIndex

	this.gcIndex++
	if this.gcIndex >= int(this.countMaps) {
		this.gcIndex = 0
	}

	this.gcLocker.Unlock()

	var currentTime = fasttime.Now().Unix()

	this.locker.RLock(gcIndex)
	var itemMap = this.itemMaps[gcIndex]
	var expiredKeys = []uint64{}
	for key, item := range itemMap {
		if item.IsExpired(currentTime) {
			expiredKeys = append(expiredKeys, key)
		}
	}
	var tooManyItems = len(itemMap) > maxItemsPerGroup // prevent too many items
	this.locker.RUnlock(gcIndex)

	if len(expiredKeys) > 0 {
		this.locker.Lock(gcIndex)
		for _, key := range expiredKeys {
			delete(itemMap, key)
		}
		this.locker.Unlock(gcIndex)
	}

	if tooManyItems {
		this.locker.Lock(gcIndex)
		var count = len(itemMap) - maxItemsPerGroup
		if count > 0 {
			itemMap = this.itemMaps[gcIndex]
			for key := range itemMap {
				delete(itemMap, key)
				count--
				if count < 0 {
					break
				}
			}
		}
		this.locker.Unlock(gcIndex)
	}
}

func (this *Counter[T]) CountMaps() int {
	return int(this.countMaps)
}

// calculate hash of the key
func (this *Counter[T]) hash(key string) uint64 {
	return xxhash.Sum64String(key)
}
