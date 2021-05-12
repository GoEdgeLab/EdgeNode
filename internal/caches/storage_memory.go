package caches

import (
	"fmt"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/errors"
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
}

type MemoryStorage struct {
	policy        *serverconfigs.HTTPCachePolicy
	list          *List
	locker        *sync.RWMutex
	valuesMap     map[uint64]*MemoryItem
	ticker        *utils.Ticker
	purgeDuration time.Duration
	totalSize     int64
}

func NewMemoryStorage(policy *serverconfigs.HTTPCachePolicy) *MemoryStorage {
	return &MemoryStorage{
		policy:    policy,
		list:      NewList(),
		locker:    &sync.RWMutex{},
		valuesMap: map[uint64]*MemoryItem{},
	}
}

// Init 初始化
func (this *MemoryStorage) Init() error {
	this.list.OnAdd(func(item *Item) {
		atomic.AddInt64(&this.totalSize, item.Size())
	})
	this.list.OnRemove(func(item *Item) {
		atomic.AddInt64(&this.totalSize, -item.Size())
	})

	if this.purgeDuration <= 0 {
		this.purgeDuration = 30 * time.Second
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
	// 检查是否超出最大值
	if this.policy.MaxKeys > 0 && this.list.Count() > this.policy.MaxKeys {
		return nil, errors.New("write memory cache failed: too many keys in cache storage")
	}
	capacityBytes := this.memoryCapacityBytes()
	if capacityBytes > 0 && capacityBytes <= this.totalSize {
		return nil, errors.New("write memory cache failed: over memory size, real size: " + strconv.FormatInt(this.totalSize, 10) + " bytes")
	}

	// 先删除
	err := this.Delete(key)
	if err != nil {
		return nil, err
	}

	return NewMemoryWriter(this.valuesMap, key, expiredAt, status, this.locker), nil
}

// Delete 删除某个键值对应的缓存
func (this *MemoryStorage) Delete(key string) error {
	hash := this.hash(key)
	this.locker.Lock()
	delete(this.valuesMap, hash)
	this.list.Remove(fmt.Sprintf("%d", hash))
	this.locker.Unlock()
	return nil
}

// Stat 统计缓存
func (this *MemoryStorage) Stat() (*Stat, error) {
	this.locker.RLock()
	defer this.locker.RUnlock()

	return this.list.Stat(func(hash string) bool {
		return true
	}), nil
}

// CleanAll 清除所有缓存
func (this *MemoryStorage) CleanAll() error {
	this.locker.Lock()
	this.valuesMap = map[uint64]*MemoryItem{}
	this.list.Reset()
	atomic.StoreInt64(&this.totalSize, 0)
	this.locker.Unlock()
	return nil
}

// Purge 批量删除缓存
func (this *MemoryStorage) Purge(keys []string, urlType string) error {
	// 目录
	if urlType == "dir" {
		resultKeys := []string{}
		for _, key := range keys {
			resultKeys = append(resultKeys, this.list.FindKeysWithPrefix(key)...)
		}
		keys = resultKeys
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
	defer this.locker.Unlock()

	this.valuesMap = map[uint64]*MemoryItem{}
	this.list.Reset()
	if this.ticker != nil {
		this.ticker.Stop()
	}
}

// Policy 获取当前存储的Policy
func (this *MemoryStorage) Policy() *serverconfigs.HTTPCachePolicy {
	return this.policy
}

// AddToList 将缓存添加到列表
func (this *MemoryStorage) AddToList(item *Item) {
	item.MetaSize = int64(len(item.Key)) + 32 /** 32是我们评估的数据结构的长度 **/
	hash := fmt.Sprintf("%d", this.hash(item.Key))
	this.list.Add(hash, item)
}

// 计算Key Hash
func (this *MemoryStorage) hash(key string) uint64 {
	return xxhash.Sum64String(key)
}

// 清理任务
func (this *MemoryStorage) purgeLoop() {
	this.list.Purge(2048, func(hash string) {
		uintHash, err := strconv.ParseUint(hash, 10, 64)
		if err == nil {
			this.locker.Lock()
			delete(this.valuesMap, uintHash)
			this.locker.Unlock()
		}
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
