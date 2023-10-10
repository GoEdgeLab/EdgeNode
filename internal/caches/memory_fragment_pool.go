// Copyright 2023 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package caches

import (
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/TeaOSLab/EdgeNode/internal/utils/fasttime"
	"github.com/iwind/TeaGo/logs"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

const (
	minMemoryFragmentPoolItemSize       = 8 << 10
	maxMemoryFragmentPoolItemSize       = 128 << 20
	maxItemsInMemoryFragmentPoolBucket  = 1024
	memoryFragmentPoolBucketSegmentSize = 512 << 10
	maxMemoryFragmentPoolItemAgeSeconds = 60
)

var SharedFragmentMemoryPool *MemoryFragmentPool

func init() {
	if !teaconst.IsMain {
		return
	}

	SharedFragmentMemoryPool = NewMemoryFragmentPool()

	goman.New(func() {
		var ticker = time.NewTicker(200 * time.Millisecond)
		for range ticker.C {
			for i := 0; i < 10; i++ { // skip N empty buckets
				var isEmpty = SharedFragmentMemoryPool.GCNextBucket()
				if !isEmpty {
					break
				}
			}
		}
	})
}

type MemoryFragmentPoolItem struct {
	Bytes []byte

	size      int64
	createdAt int64

	Refs int32
}

func (this *MemoryFragmentPoolItem) IsExpired() bool {
	return this.createdAt < fasttime.Now().Unix()-maxMemoryFragmentPoolItemAgeSeconds
}

func (this *MemoryFragmentPoolItem) Reset() {
	this.Bytes = nil
}

func (this *MemoryFragmentPoolItem) IsAvailable() bool {
	return atomic.AddInt32(&this.Refs, 1) == 1
}

// MemoryFragmentPool memory fragments management
type MemoryFragmentPool struct {
	bucketMaps    []map[uint64]*MemoryFragmentPoolItem // [  { id => Zero }, ... ]
	countBuckets  int
	gcBucketIndex int

	mu sync.RWMutex

	id          uint64
	totalMemory int64

	isOk     bool
	capacity int64

	debugMode bool
	countGet  uint64
	countNew  uint64
}

// NewMemoryFragmentPool create new fragment memory pool
func NewMemoryFragmentPool() *MemoryFragmentPool {
	var pool = &MemoryFragmentPool{}
	pool.init()
	return pool
}

func (this *MemoryFragmentPool) init() {
	var capacity = int64(utils.SystemMemoryGB()) << 30 / 16
	if capacity > 256<<20 {
		this.isOk = true
		this.capacity = capacity

		this.bucketMaps = []map[uint64]*MemoryFragmentPoolItem{}
		for i := 0; i < maxMemoryFragmentPoolItemSize/memoryFragmentPoolBucketSegmentSize+1; i++ {
			this.bucketMaps = append(this.bucketMaps, map[uint64]*MemoryFragmentPoolItem{})
		}
		this.countBuckets = len(this.bucketMaps)
	}

	// print statistics for debug
	if len(os.Getenv("GOEDGE_DEBUG_MEMORY_FRAGMENT_POOL")) > 0 {
		this.debugMode = true

		go func() {
			var maxRounds = 10_000
			var ticker = time.NewTicker(10 * time.Second)
			for range ticker.C {
				logs.Println("reused:", this.countGet, "created:", this.countNew, "fragments:", this.Len(), "memory:", this.totalMemory>>20, "MB")

				maxRounds--
				if maxRounds <= 0 {
					break
				}
			}
		}()
	}
}

// Get try to get a bytes object
func (this *MemoryFragmentPool) Get(expectSize int64) (resultBytes []byte, ok bool) {
	if !this.isOk {
		return
	}

	if expectSize <= 0 {
		return
	}

	// DO NOT check min segment size

	this.mu.RLock()

	var bucketIndex = this.bucketIndexForSize(expectSize)
	var resultItemId uint64
	const maxSearchingBuckets = 20
	for i := bucketIndex; i <= bucketIndex+maxSearchingBuckets; i++ {
		resultBytes, resultItemId, ok = this.findItemInMap(this.bucketMaps[i], expectSize)
		if ok {
			this.mu.RUnlock()

			// remove from bucket
			this.mu.Lock()
			delete(this.bucketMaps[i], resultItemId)
			this.mu.Unlock()

			return
		}
		if i >= this.countBuckets {
			break
		}
	}
	this.mu.RUnlock()

	return
}

// Put a bytes object to specified bucket
func (this *MemoryFragmentPool) Put(data []byte) (ok bool) {
	if !this.isOk {
		return
	}

	var l = int64(cap(data)) // MUST be 'cap' instead of 'len'

	if l < minMemoryFragmentPoolItemSize || l > maxMemoryFragmentPoolItemSize {
		return
	}

	if atomic.LoadInt64(&this.totalMemory) >= this.capacity {
		return
	}

	var itemId = atomic.AddUint64(&this.id, 1)

	this.mu.Lock()
	defer this.mu.Unlock()

	var bucketMap = this.bucketMaps[this.bucketIndexForSize(l)]
	if len(bucketMap) >= maxItemsInMemoryFragmentPoolBucket {
		return
	}

	atomic.AddInt64(&this.totalMemory, l)

	bucketMap[itemId] = &MemoryFragmentPoolItem{
		Bytes:     data,
		size:      l,
		createdAt: fasttime.Now().Unix(),
	}

	return true
}

// GC fully GC
func (this *MemoryFragmentPool) GC() {
	if !this.isOk {
		return
	}

	var totalMemory = atomic.LoadInt64(&this.totalMemory)
	if totalMemory < this.capacity {
		return
	}

	this.mu.Lock()
	defer this.mu.Unlock()

	var garbageSize = totalMemory * 1 / 10 // 10%

	// remove expired
	for _, bucketMap := range this.bucketMaps {
		for itemId, item := range bucketMap {
			if item.IsExpired() {
				delete(bucketMap, itemId)
				item.Reset()
				atomic.AddInt64(&this.totalMemory, -item.size)

				garbageSize -= item.size
			}
		}
	}

	// remove others
	if garbageSize > 0 {
		for _, bucketMap := range this.bucketMaps {
			for itemId, item := range bucketMap {
				delete(bucketMap, itemId)
				item.Reset()
				atomic.AddInt64(&this.totalMemory, -item.size)

				garbageSize -= item.size
				if garbageSize <= 0 {
					break
				}
			}
		}
	}
}

// GCNextBucket gc one bucket
func (this *MemoryFragmentPool) GCNextBucket() (isEmpty bool) {
	if !this.isOk {
		return
	}

	var itemIds = []uint64{}

	// find
	this.mu.RLock()

	var bucketIndex = this.gcBucketIndex
	var bucketMap = this.bucketMaps[bucketIndex]
	isEmpty = len(bucketMap) == 0
	if isEmpty {
		this.mu.RUnlock()

		// move to next bucket index
		bucketIndex++
		if bucketIndex >= this.countBuckets {
			bucketIndex = 0
		}
		this.gcBucketIndex = bucketIndex

		return
	}

	for itemId, item := range bucketMap {
		if item.IsExpired() {
			itemIds = append(itemIds, itemId)
		}
	}

	this.mu.RUnlock()

	// remove
	if len(itemIds) > 0 {
		this.mu.Lock()
		for _, itemId := range itemIds {
			item, ok := bucketMap[itemId]
			if !ok {
				continue
			}
			if !item.IsAvailable() {
				continue
			}
			delete(bucketMap, itemId)
			item.Reset()
			atomic.AddInt64(&this.totalMemory, -item.size)
		}
		this.mu.Unlock()
	}

	// move to next bucket index
	bucketIndex++
	if bucketIndex >= this.countBuckets {
		bucketIndex = 0
	}
	this.gcBucketIndex = bucketIndex

	return
}

func (this *MemoryFragmentPool) SetCapacity(capacity int64) {
	this.capacity = capacity
}

func (this *MemoryFragmentPool) TotalSize() int64 {
	return atomic.LoadInt64(&this.totalMemory)
}

func (this *MemoryFragmentPool) Len() int {
	this.mu.Lock()
	defer this.mu.Unlock()
	var count = 0
	for _, bucketMap := range this.bucketMaps {
		count += len(bucketMap)
	}
	return count
}

func (this *MemoryFragmentPool) IncreaseNew() {
	if this.isOk && this.debugMode {
		atomic.AddUint64(&this.countNew, 1)
	}
}

func (this *MemoryFragmentPool) bucketIndexForSize(size int64) int {
	return int(size / memoryFragmentPoolBucketSegmentSize)
}

func (this *MemoryFragmentPool) findItemInMap(bucketMap map[uint64]*MemoryFragmentPoolItem, expectSize int64) (resultBytes []byte, resultItemId uint64, ok bool) {
	if len(bucketMap) == 0 {
		return
	}

	for itemId, item := range bucketMap {
		if item.size >= expectSize {
			// check if is referred
			if !item.IsAvailable() {
				continue
			}

			// return result
			if item.size != expectSize {
				resultBytes = item.Bytes[:expectSize]
			} else {
				resultBytes = item.Bytes
			}

			// reset old item
			item.Reset()
			atomic.AddInt64(&this.totalMemory, -item.size)

			resultItemId = itemId

			if this.debugMode {
				atomic.AddUint64(&this.countGet, 1)
			}

			ok = true

			return
		}
	}

	return
}
