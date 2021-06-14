package caches

import (
	"fmt"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/cespare/xxhash"
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
	policy        *serverconfigs.HTTPCachePolicy
	list          ListInterface
	locker        *sync.RWMutex
	valuesMap     map[uint64]*MemoryItem
	ticker        *utils.Ticker
	purgeDuration time.Duration
	totalSize     int64
	writingKeyMap map[string]bool // key => bool
}

func NewMemoryStorage(policy *serverconfigs.HTTPCachePolicy) *MemoryStorage {
	return &MemoryStorage{
		policy:        policy,
		list:          NewMemoryList(),
		locker:        &sync.RWMutex{},
		valuesMap:     map[uint64]*MemoryItem{},
		writingKeyMap: map[string]bool{},
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

	if this.purgeDuration <= 0 {
		this.purgeDuration = 10 * time.Second
	}

	// 启动定时清理任务
	this.ticker = utils.NewTicker(this.purgeDuration)
	go func() {
		for this.ticker.Next() {
			this.purgeLoop()
		}
	}()

	return nil
}

// OpenReader 读取缓存
func (this *MemoryStorage) OpenReader(key string) (Reader, error) {
	hash := this.hash(key)

	this.locker.RLock()
	defer this.locker.RUnlock()

	item := this.valuesMap[hash]
	if item == nil || !item.IsDone {
		return nil, ErrNotFound
	}

	if item.ExpiredAt > utils.UnixTime() {
		reader := NewMemoryReader(item)
		err := reader.Init()
		if err != nil {
			return nil, err
		}
		return reader, nil
	}

	_ = this.Delete(key)

	return nil, ErrNotFound
}

// OpenWriter 打开缓存写入器等待写入
func (this *MemoryStorage) OpenWriter(key string, expiredAt int64, status int) (Writer, error) {
	this.locker.Lock()
	defer this.locker.Unlock()

	// 是否正在写入
	var isWriting = false
	_, ok := this.writingKeyMap[key]
	if ok {
		return nil, ErrFileIsWriting
	}
	this.writingKeyMap[key] = true
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
	if capacityBytes > 0 && capacityBytes <= this.totalSize {
		return nil, NewCapacityError("write memory cache failed: over memory size: " + strconv.FormatInt(capacityBytes, 10) + ", current size: " + strconv.FormatInt(this.totalSize, 10) + " bytes")
	}

	// 先删除
	err = this.deleteWithoutKey(key)
	if err != nil {
		return nil, err
	}

	isWriting = true
	return NewMemoryWriter(this.valuesMap, key, expiredAt, status, this.locker, func() {
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
	this.writingKeyMap = map[string]bool{}
	_ = this.list.Reset()
	if this.ticker != nil {
		this.ticker.Stop()
	}

	_ = this.list.Close()

	this.locker.Unlock()

	remotelogs.Println("CACHE", "close memory storage '"+strconv.FormatInt(this.policy.Id, 10)+"'")
}

// Policy 获取当前存储的Policy
func (this *MemoryStorage) Policy() *serverconfigs.HTTPCachePolicy {
	return this.policy
}

// AddToList 将缓存添加到列表
func (this *MemoryStorage) AddToList(item *Item) {
	item.MetaSize = int64(len(item.Key)) + 32 /** 32是我们评估的数据结构的长度 **/
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

// 计算Key Hash
func (this *MemoryStorage) hash(key string) uint64 {
	return xxhash.Sum64String(key)
}

// 清理任务
func (this *MemoryStorage) purgeLoop() {
	_ = this.list.Purge(2048, func(hash string) error {
		uintHash, err := strconv.ParseUint(hash, 10, 64)
		if err == nil {
			this.locker.Lock()
			delete(this.valuesMap, uintHash)
			this.locker.Unlock()
		}
		return nil
	})
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

func (this *MemoryStorage) deleteWithoutKey(key string) error {
	hash := this.hash(key)
	delete(this.valuesMap, hash)
	_ = this.list.Remove(fmt.Sprintf("%d", hash))
	return nil
}
