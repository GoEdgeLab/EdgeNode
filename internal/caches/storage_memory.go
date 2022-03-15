package caches

import (
	"fmt"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/trackers"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	setutils "github.com/TeaOSLab/EdgeNode/internal/utils/sets"
	"github.com/TeaOSLab/EdgeNode/internal/utils/sizes"
	"github.com/TeaOSLab/EdgeNode/internal/zero"
	"github.com/cespare/xxhash"
	"github.com/iwind/TeaGo/rands"
	"github.com/iwind/TeaGo/types"
	"math"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type MemoryItem struct {
	ExpiredAt   int64
	HeaderValue []byte
	BodyValue   []byte
	Status      int
	IsDone      bool
	ModifiedAt  int64
}

func (this *MemoryItem) IsExpired() bool {
	return this.ExpiredAt < utils.UnixTime()
}

type MemoryStorage struct {
	parentStorage StorageInterface

	policy *serverconfigs.HTTPCachePolicy
	list   ListInterface
	locker *sync.RWMutex

	valuesMap map[uint64]*MemoryItem // hash => item

	dirtyChan      chan string // hash chan
	dirtyQueueSize int

	purgeTicker *utils.Ticker

	totalSize     int64
	writingKeyMap map[string]zero.Zero // key => bool

	ignoreKeys *setutils.FixedSet
}

func NewMemoryStorage(policy *serverconfigs.HTTPCachePolicy, parentStorage StorageInterface) *MemoryStorage {
	var dirtyChan chan string
	var queueSize = policy.MemoryAutoFlushQueueSize

	if parentStorage != nil {
		if queueSize <= 0 {
			queueSize = 2048 + int(policy.CapacityBytes()/sizes.G)*2048
		}

		dirtyChan = make(chan string, queueSize)
	}
	return &MemoryStorage{
		parentStorage:  parentStorage,
		policy:         policy,
		list:           NewMemoryList(),
		locker:         &sync.RWMutex{},
		valuesMap:      map[uint64]*MemoryItem{},
		dirtyChan:      dirtyChan,
		dirtyQueueSize: queueSize,
		writingKeyMap:  map[string]zero.Zero{},
		ignoreKeys:     setutils.NewFixedSet(32768),
	}
}

// Init 初始化
func (this *MemoryStorage) Init() error {
	_ = this.list.Init()

	this.list.OnAdd(func(item *Item) {
		atomic.AddInt64(&this.totalSize, item.TotalSize())
	})
	this.list.OnRemove(func(item *Item) {
		atomic.AddInt64(&this.totalSize, -item.TotalSize())
	})

	this.initPurgeTicker()

	// 启动定时Flush memory to disk任务
	if this.parentStorage != nil {
		var threads = runtime.NumCPU()
		if threads == 0 {
			threads = 1
		} else if threads > 8 {
			threads = 8
		}
		for i := 0; i < threads; i++ {
			goman.New(func() {
				for hash := range this.dirtyChan {
					this.flushItem(hash)
				}
			})
		}
	}

	return nil
}

// OpenReader 读取缓存
func (this *MemoryStorage) OpenReader(key string, useStale bool, isPartial bool) (Reader, error) {
	hash := this.hash(key)

	this.locker.RLock()
	item := this.valuesMap[hash]
	if item == nil || !item.IsDone {
		this.locker.RUnlock()
		return nil, ErrNotFound
	}

	if useStale || (item.ExpiredAt > utils.UnixTime()) {
		reader := NewMemoryReader(item)
		err := reader.Init()
		if err != nil {
			this.locker.RUnlock()
			return nil, err
		}
		this.locker.RUnlock()

		// 增加点击量
		// 1/1000采样
		// TODO 考虑是否在缓存策略里设置
		if rands.Int(0, 1000) == 0 {
			var hitErr = this.list.IncreaseHit(types.String(hash))
			if hitErr != nil {
				// 此错误可以忽略
				remotelogs.Error("CACHE", "increase hit failed: "+hitErr.Error())
			}
		}

		return reader, nil
	}
	this.locker.RUnlock()

	_ = this.Delete(key)

	return nil, ErrNotFound
}

// OpenWriter 打开缓存写入器等待写入
func (this *MemoryStorage) OpenWriter(key string, expiredAt int64, status int, size int64, maxSize int64, isPartial bool) (Writer, error) {
	if this.ignoreKeys.Has(key) {
		return nil, ErrEntityTooLarge
	}

	// TODO 内存缓存暂时不支持分块内容存储
	if isPartial {
		return nil, ErrFileIsWriting
	}
	return this.openWriter(key, expiredAt, status, size, maxSize, true)
}

func (this *MemoryStorage) openWriter(key string, expiredAt int64, status int, size int64, maxSize int64, isDirty bool) (Writer, error) {
	// 待写入队列是否已满
	if isDirty &&
		this.parentStorage != nil &&
		this.dirtyQueueSize > 0 &&
		len(this.dirtyChan) == this.dirtyQueueSize &&
		(expiredAt <= 0 || expiredAt > time.Now().Unix()+7200) { // 缓存时间过长
		return nil, ErrWritingQueueFull
	}

	this.locker.Lock()
	defer this.locker.Unlock()

	// 是否正在写入
	var isWriting = false
	_, ok := this.writingKeyMap[key]
	if ok {
		return nil, ErrFileIsWriting
	}
	this.writingKeyMap[key] = zero.New()
	defer func() {
		if !isWriting {
			delete(this.writingKeyMap, key)
		}
	}()

	// 检查是否过期
	hash := this.hash(key)
	item, ok := this.valuesMap[hash]
	if ok && !item.IsExpired() {
		return nil, ErrFileIsWriting
	}

	// 检查是否超出最大值
	totalKeys, err := this.list.Count()
	if err != nil {
		return nil, err
	}
	if this.policy.MaxKeys > 0 && totalKeys > this.policy.MaxKeys {
		return nil, NewCapacityError("write memory cache failed: too many keys in cache storage")
	}
	capacityBytes := this.memoryCapacityBytes()
	if size < 0 {
		size = 0
	}
	if capacityBytes > 0 && capacityBytes <= this.totalSize+size {
		return nil, NewCapacityError("write memory cache failed: over memory size: " + strconv.FormatInt(capacityBytes, 10) + ", current size: " + strconv.FormatInt(this.totalSize, 10) + " bytes")
	}

	// 先删除
	err = this.deleteWithoutLocker(key)
	if err != nil {
		return nil, err
	}

	isWriting = true
	return NewMemoryWriter(this, key, expiredAt, status, isDirty, maxSize, func() {
		this.locker.Lock()
		delete(this.writingKeyMap, key)
		this.locker.Unlock()
	}), nil
}

// Delete 删除某个键值对应的缓存
func (this *MemoryStorage) Delete(key string) error {
	hash := this.hash(key)
	this.locker.Lock()
	delete(this.valuesMap, hash)
	_ = this.list.Remove(fmt.Sprintf("%d", hash))
	this.locker.Unlock()
	return nil
}

// Stat 统计缓存
func (this *MemoryStorage) Stat() (*Stat, error) {
	this.locker.RLock()
	defer this.locker.RUnlock()

	return this.list.Stat(func(hash string) bool {
		return true
	})
}

// CleanAll 清除所有缓存
func (this *MemoryStorage) CleanAll() error {
	this.locker.Lock()
	this.valuesMap = map[uint64]*MemoryItem{}
	_ = this.list.Reset()
	atomic.StoreInt64(&this.totalSize, 0)
	this.locker.Unlock()
	return nil
}

// Purge 批量删除缓存
func (this *MemoryStorage) Purge(keys []string, urlType string) error {
	// 目录
	if urlType == "dir" {
		for _, key := range keys {
			err := this.list.CleanPrefix(key)
			if err != nil {
				return err
			}
		}
	}

	for _, key := range keys {
		err := this.Delete(key)
		if err != nil {
			return err
		}
	}
	return nil
}

// Stop 停止缓存策略
func (this *MemoryStorage) Stop() {
	this.locker.Lock()

	this.valuesMap = map[uint64]*MemoryItem{}
	this.writingKeyMap = map[string]zero.Zero{}
	_ = this.list.Reset()
	if this.purgeTicker != nil {
		this.purgeTicker.Stop()
	}

	if this.dirtyChan != nil {
		close(this.dirtyChan)
	}

	_ = this.list.Close()

	this.locker.Unlock()

	this.ignoreKeys.Reset()

	// 回收内存
	runtime.GC()

	remotelogs.Println("CACHE", "close memory storage '"+strconv.FormatInt(this.policy.Id, 10)+"'")
}

// Policy 获取当前存储的Policy
func (this *MemoryStorage) Policy() *serverconfigs.HTTPCachePolicy {
	return this.policy
}

// UpdatePolicy 修改策略
func (this *MemoryStorage) UpdatePolicy(newPolicy *serverconfigs.HTTPCachePolicy) {
	var oldPolicy = this.policy
	this.policy = newPolicy

	if oldPolicy.MemoryAutoPurgeInterval != newPolicy.MemoryAutoPurgeInterval {
		this.initPurgeTicker()
	}

	// 如果是空的，则清空
	if newPolicy.CapacityBytes() == 0 {
		_ = this.CleanAll()
	}
}

// CanUpdatePolicy 检查策略是否可以更新
func (this *MemoryStorage) CanUpdatePolicy(newPolicy *serverconfigs.HTTPCachePolicy) bool {
	return true
}

// AddToList 将缓存添加到列表
func (this *MemoryStorage) AddToList(item *Item) {
	item.MetaSize = int64(len(item.Key)) + 128 /** 128是我们评估的数据结构的长度 **/
	hash := fmt.Sprintf("%d", this.hash(item.Key))
	_ = this.list.Add(hash, item)
}

// TotalDiskSize 消耗的磁盘尺寸
func (this *MemoryStorage) TotalDiskSize() int64 {
	return 0
}

// TotalMemorySize 内存尺寸
func (this *MemoryStorage) TotalMemorySize() int64 {
	return atomic.LoadInt64(&this.totalSize)
}

// IgnoreKey 忽略某个Key，即不缓存某个Key
func (this *MemoryStorage) IgnoreKey(key string) {
	this.ignoreKeys.Push(key)
}

// 计算Key Hash
func (this *MemoryStorage) hash(key string) uint64 {
	return xxhash.Sum64String(key)
}

// 清理任务
func (this *MemoryStorage) purgeLoop() {
	// 计算是否应该开启LFU清理
	var capacityBytes = this.policy.CapacityBytes()
	var startLFU = false
	var usedPercent = float32(this.TotalMemorySize()*100) / float32(capacityBytes)
	var lfuFreePercent = this.policy.MemoryLFUFreePercent
	if lfuFreePercent <= 0 {
		lfuFreePercent = 5
	}
	if capacityBytes > 0 {
		if lfuFreePercent < 100 {
			if usedPercent >= 100-lfuFreePercent {
				startLFU = true
			}
		}
	}

	// 清理过期
	var purgeCount = this.policy.MemoryAutoPurgeCount
	if purgeCount <= 0 {
		purgeCount = 2000
	}
	_, _ = this.list.Purge(purgeCount, func(hash string) error {
		uintHash, err := strconv.ParseUint(hash, 10, 64)
		if err == nil {
			this.locker.Lock()
			delete(this.valuesMap, uintHash)
			this.locker.Unlock()
		}
		return nil
	})

	// LFU
	if startLFU {
		var total, _ = this.list.Count()
		if total > 0 {
			var count = types.Int(math.Ceil(float64(total) * float64(lfuFreePercent*2) / 100))
			if count > 0 {
				// 限制单次清理的条数，防止占用太多系统资源
				if count > 2000 {
					count = 2000
				}

				// 这里不提示LFU，因为此事件将会非常频繁

				err := this.list.PurgeLFU(count, func(hash string) error {
					uintHash, err := strconv.ParseUint(hash, 10, 64)
					if err == nil {
						this.locker.Lock()
						delete(this.valuesMap, uintHash)
						this.locker.Unlock()
					}
					return nil
				})
				if err != nil {
					remotelogs.Warn("CACHE", "purge memory storage in LFU failed: "+err.Error())
				}
			}
		}
	}
}

// Flush任务
func (this *MemoryStorage) flushItem(key string) {
	if this.parentStorage == nil {
		return
	}
	var hash = this.hash(key)

	this.locker.RLock()
	item, ok := this.valuesMap[hash]
	this.locker.RUnlock()

	if !ok {
		return
	}
	if !item.IsDone || item.IsExpired() {
		return
	}

	writer, err := this.parentStorage.OpenWriter(key, item.ExpiredAt, item.Status, -1, -1, false)
	if err != nil {
		if !CanIgnoreErr(err) {
			remotelogs.Error("CACHE", "flush items failed: open writer failed: "+err.Error())
		}
		return
	}

	_, err = writer.WriteHeader(item.HeaderValue)
	if err != nil {
		_ = writer.Discard()
		remotelogs.Error("CACHE", "flush items failed: write header failed: "+err.Error())
		return
	}

	_, err = writer.Write(item.BodyValue)
	if err != nil {
		_ = writer.Discard()
		remotelogs.Error("CACHE", "flush items failed: writer body failed: "+err.Error())
		return
	}

	err = writer.Close()
	if err != nil {
		_ = writer.Discard()
		remotelogs.Error("CACHE", "flush items failed: close writer failed: "+err.Error())
		return
	}

	this.parentStorage.AddToList(&Item{
		Type:       writer.ItemType(),
		Key:        key,
		ExpiredAt:  item.ExpiredAt,
		HeaderSize: writer.HeaderSize(),
		BodySize:   writer.BodySize(),
	})

	// 从内存中移除
	_ = this.Delete(key)

	return
}

func (this *MemoryStorage) memoryCapacityBytes() int64 {
	if this.policy == nil {
		return 0
	}
	c1 := int64(0)
	if this.policy.Capacity != nil {
		c1 = this.policy.Capacity.Bytes()
	}
	if SharedManager.MaxMemoryCapacity != nil {
		c2 := SharedManager.MaxMemoryCapacity.Bytes()
		if c2 > 0 {
			return c2
		}
	}
	return c1
}

func (this *MemoryStorage) deleteWithoutLocker(key string) error {
	hash := this.hash(key)
	delete(this.valuesMap, hash)
	_ = this.list.Remove(fmt.Sprintf("%d", hash))
	return nil
}

func (this *MemoryStorage) initPurgeTicker() {
	var autoPurgeInterval = this.policy.MemoryAutoPurgeInterval
	if autoPurgeInterval <= 0 {
		autoPurgeInterval = 5
	}

	// 启动定时清理任务

	if this.purgeTicker != nil {
		this.purgeTicker.Stop()
	}

	this.purgeTicker = utils.NewTicker(time.Duration(autoPurgeInterval) * time.Second)
	goman.New(func() {
		for this.purgeTicker.Next() {
			var tr = trackers.Begin("MEMORY_CACHE_STORAGE_PURGE_LOOP")
			this.purgeLoop()
			tr.End()
		}
	})
}
