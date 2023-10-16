package caches

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/trackers"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/TeaOSLab/EdgeNode/internal/utils/fasttime"
	fsutils "github.com/TeaOSLab/EdgeNode/internal/utils/fs"
	setutils "github.com/TeaOSLab/EdgeNode/internal/utils/sets"
	"github.com/TeaOSLab/EdgeNode/internal/zero"
	"github.com/cespare/xxhash"
	"github.com/iwind/TeaGo/types"
	"math"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type MemoryItem struct {
	ExpiresAt   int64
	HeaderValue []byte
	BodyValue   []byte
	Status      int
	IsDone      bool
	ModifiedAt  int64

	IsPrepared  bool
	WriteOffset int64

	isReferring bool // if it is referring by other objects
}

func (this *MemoryItem) IsExpired() bool {
	return this.ExpiresAt < fasttime.Now().Unix()
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

	usedSize      int64
	writingKeyMap map[string]zero.Zero // key => bool

	ignoreKeys *setutils.FixedSet
}

func NewMemoryStorage(policy *serverconfigs.HTTPCachePolicy, parentStorage StorageInterface) *MemoryStorage {
	var dirtyChan chan string
	var queueSize = policy.MemoryAutoFlushQueueSize

	if parentStorage != nil {
		if queueSize <= 0 {
			queueSize = utils.SystemMemoryGB() * 100_000
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
		atomic.AddInt64(&this.usedSize, item.TotalSize())
	})
	this.list.OnRemove(func(item *Item) {
		atomic.AddInt64(&this.usedSize, -item.TotalSize())
	})

	this.initPurgeTicker()

	// 启动定时Flush memory to disk任务
	if this.parentStorage != nil {
		// TODO 应该根据磁盘性能决定线程数
		// TODO 线程数应该可以在缓存策略和节点中设定
		var threads = runtime.NumCPU()

		for i := 0; i < threads; i++ {
			goman.New(func() {
				this.startFlush()
			})
		}
	}

	return nil
}

// OpenReader 读取缓存
func (this *MemoryStorage) OpenReader(key string, useStale bool, isPartial bool) (Reader, error) {
	var hash = this.hash(key)

	// check if exists in list
	exists, _ := this.list.Exist(types.String(hash))
	if !exists {
		return nil, ErrNotFound
	}

	// read from valuesMap
	this.locker.RLock()
	var item = this.valuesMap[hash]

	if item != nil {
		item.isReferring = true
	}

	if item == nil || !item.IsDone {
		this.locker.RUnlock()
		return nil, ErrNotFound
	}

	if useStale || (item.ExpiresAt > fasttime.Now().Unix()) {
		var reader = NewMemoryReader(item)
		err := reader.Init()
		if err != nil {
			this.locker.RUnlock()
			return nil, err
		}
		this.locker.RUnlock()

		return reader, nil
	}
	this.locker.RUnlock()

	_ = this.Delete(key)

	return nil, ErrNotFound
}

// OpenWriter 打开缓存写入器等待写入
func (this *MemoryStorage) OpenWriter(key string, expiredAt int64, status int, headerSize int, bodySize int64, maxSize int64, isPartial bool) (Writer, error) {
	if maxSize > 0 && this.ignoreKeys.Has(types.String(maxSize)+"$"+key) {
		return nil, ErrEntityTooLarge
	}

	// TODO 内存缓存暂时不支持分块内容存储
	if isPartial {
		return nil, ErrFileIsWriting
	}
	return this.openWriter(key, expiredAt, status, headerSize, bodySize, maxSize, true)
}

// OpenFlushWriter 打开从其他媒介直接刷入的写入器
func (this *MemoryStorage) OpenFlushWriter(key string, expiresAt int64, status int, headerSize int, bodySize int64) (Writer, error) {
	return this.openWriter(key, expiresAt, status, headerSize, bodySize, -1, true)
}

func (this *MemoryStorage) openWriter(key string, expiresAt int64, status int, headerSize int, bodySize int64, maxSize int64, isDirty bool) (Writer, error) {
	// 待写入队列是否已满
	if isDirty &&
		this.parentStorage != nil &&
		this.dirtyQueueSize > 0 &&
		len(this.dirtyChan) >= this.dirtyQueueSize-int(fsutils.DiskMaxWrites) /** delta **/ { // 缓存时间过长
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
	var hash = this.hash(key)
	item, ok := this.valuesMap[hash]
	if ok && !item.IsExpired() {
		var hashString = types.String(hash)
		exists, _ := this.list.Exist(hashString)
		if !exists {
			// remove from values map
			delete(this.valuesMap, hash)
			_ = this.list.Remove(hashString)
			item = nil
		} else {
			return nil, ErrFileIsWriting
		}
	}

	// 检查是否超出容量最大值
	var capacityBytes = this.memoryCapacityBytes()
	if bodySize < 0 {
		bodySize = 0
	}
	if capacityBytes > 0 && capacityBytes <= atomic.LoadInt64(&this.usedSize)+bodySize {
		return nil, NewCapacityError("write memory cache failed: over memory size: " + strconv.FormatInt(capacityBytes, 10) + ", current size: " + strconv.FormatInt(this.usedSize, 10) + " bytes")
	}

	// 先删除
	err := this.deleteWithoutLocker(key)
	if err != nil {
		return nil, err
	}

	isWriting = true
	return NewMemoryWriter(this, key, expiresAt, status, isDirty, bodySize, maxSize, func(valueItem *MemoryItem) {
		this.locker.Lock()
		delete(this.writingKeyMap, key)
		this.locker.Unlock()
	}), nil
}

// Delete 删除某个键值对应的缓存
func (this *MemoryStorage) Delete(key string) error {
	var hash = this.hash(key)
	this.locker.Lock()
	delete(this.valuesMap, hash)
	_ = this.list.Remove(types.String(hash))
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
	atomic.StoreInt64(&this.usedSize, 0)
	this.locker.Unlock()
	return nil
}

// Purge 批量删除缓存
func (this *MemoryStorage) Purge(keys []string, urlType string) error {
	// 目录
	if urlType == "dir" {
		for _, key := range keys {
			// 检查是否有通配符 http(s)://*.example.com
			var schemeIndex = strings.Index(key, "://")
			if schemeIndex > 0 {
				var keyRight = key[schemeIndex+3:]
				if strings.HasPrefix(keyRight, "*.") {
					err := this.list.CleanMatchPrefix(key)
					if err != nil {
						return err
					}
					continue
				}
			}

			err := this.list.CleanPrefix(key)
			if err != nil {
				return err
			}
		}
		return nil
	}

	// URL
	for _, key := range keys {
		// 检查是否有通配符 http(s)://*.example.com
		var schemeIndex = strings.Index(key, "://")
		if schemeIndex > 0 {
			var keyRight = key[schemeIndex+3:]
			if strings.HasPrefix(keyRight, "*.") {
				err := this.list.CleanMatchKey(key)
				if err != nil {
					return err
				}
				continue
			}
		}

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

	// reset ignored keys
	this.ignoreKeys.Reset()
}

// CanUpdatePolicy 检查策略是否可以更新
func (this *MemoryStorage) CanUpdatePolicy(newPolicy *serverconfigs.HTTPCachePolicy) bool {
	return true
}

// AddToList 将缓存添加到列表
func (this *MemoryStorage) AddToList(item *Item) {
	// skip added item
	if item.MetaSize > 0 {
		return
	}

	item.MetaSize = int64(len(item.Key)) + 128 /** 128是我们评估的数据结构的长度 **/
	var hash = types.String(this.hash(item.Key))

	if len(item.Host) == 0 {
		item.Host = ParseHost(item.Key)
	}

	_ = this.list.Add(hash, item)
}

// TotalDiskSize 消耗的磁盘尺寸
func (this *MemoryStorage) TotalDiskSize() int64 {
	return 0
}

// TotalMemorySize 内存尺寸
func (this *MemoryStorage) TotalMemorySize() int64 {
	return atomic.LoadInt64(&this.usedSize)
}

// IgnoreKey 忽略某个Key，即不缓存某个Key
func (this *MemoryStorage) IgnoreKey(key string, maxSize int64) {
	this.ignoreKeys.Push(types.String(maxSize) + "$" + key)
}

// CanSendfile 是否支持Sendfile
func (this *MemoryStorage) CanSendfile() bool {
	return false
}

// HasFreeSpaceForHotItems 是否有足够的空间提供给热门内容
func (this *MemoryStorage) HasFreeSpaceForHotItems() bool {
	return atomic.LoadInt64(&this.usedSize) < this.memoryCapacityBytes()*3/4
}

// 计算Key Hash
func (this *MemoryStorage) hash(key string) uint64 {
	return xxhash.Sum64String(key)
}

// 清理任务
func (this *MemoryStorage) purgeLoop() {
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

// 开始Flush任务
func (this *MemoryStorage) startFlush() {
	var statCount = 0

	for key := range this.dirtyChan {
		statCount++

		if statCount == 100 {
			statCount = 0
		}

		this.flushItem(key)

		if fsutils.IsInExtremelyHighLoad {
			time.Sleep(1 * time.Second)
		} else if fsutils.IsInHighLoad {
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// 单次Flush任务
func (this *MemoryStorage) flushItem(key string) {
	if this.parentStorage == nil {
		return
	}
	var hash = this.hash(key)

	this.locker.RLock()
	item, ok := this.valuesMap[hash]
	this.locker.RUnlock()

	// 从内存中移除，并确保无论如何都会执行
	defer func() {
		_ = this.Delete(key)

		// 重用内存，前提是确保内存不再被引用
		if ok && item.IsDone && !item.isReferring && len(item.BodyValue) > 0 {
			SharedFragmentMemoryPool.Put(item.BodyValue)
		}
	}()

	if !ok {
		return
	}

	if !item.IsDone {
		remotelogs.Error("CACHE", "flush items failed: open writer failed: item has not been done")
		return
	}
	if item.IsExpired() {
		return
	}

	// 检查是否在列表中，防止未加入列表时就开始flush
	isInList, err := this.list.Exist(types.String(hash))
	if err != nil {
		remotelogs.Error("CACHE", "flush items failed: "+err.Error())
		return
	}
	if !isInList {
		time.Sleep(1 * time.Second)
	}

	writer, err := this.parentStorage.OpenFlushWriter(key, item.ExpiresAt, item.Status, len(item.HeaderValue), int64(len(item.BodyValue)))
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
		Host:       ParseHost(key),
		ExpiredAt:  item.ExpiresAt,
		HeaderSize: writer.HeaderSize(),
		BodySize:   writer.BodySize(),
	})
}

func (this *MemoryStorage) memoryCapacityBytes() int64 {
	var maxSystemBytes = int64(utils.SystemMemoryBytes()) / 3 // 1/3 of the system memory

	if this.policy == nil {
		return maxSystemBytes
	}

	if SharedManager.MaxMemoryCapacity != nil {
		var capacityBytes = SharedManager.MaxMemoryCapacity.Bytes()
		if capacityBytes > 0 {
			if capacityBytes > maxSystemBytes {
				return maxSystemBytes
			}

			return capacityBytes
		}
	}

	var capacity = this.policy.Capacity // copy
	if capacity != nil {
		var capacityBytes = capacity.Bytes()
		if capacityBytes > 0 {
			if capacityBytes > maxSystemBytes {
				return maxSystemBytes
			}
			return capacityBytes
		}
	}

	// 1/4 of the system memory
	return maxSystemBytes
}

func (this *MemoryStorage) deleteWithoutLocker(key string) error {
	hash := this.hash(key)
	delete(this.valuesMap, hash)
	_ = this.list.Remove(types.String(hash))
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
