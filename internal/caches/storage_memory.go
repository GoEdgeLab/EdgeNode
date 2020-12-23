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
	ExpiredAt int64
	Value     []byte
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

// 初始化
func (this *MemoryStorage) Init() error {
	this.list.OnAdd(func(item *Item) {
		atomic.AddInt64(&this.totalSize, item.Size)
	})
	this.list.OnRemove(func(item *Item) {
		atomic.AddInt64(&this.totalSize, -item.Size)
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

// 读取缓存
func (this *MemoryStorage) Read(key string, readerBuf []byte, callback func(data []byte, size int64, expiredAt int64, isEOF bool)) error {
	hash := this.hash(key)

	this.locker.RLock()
	item := this.valuesMap[hash]
	if item == nil {
		this.locker.RUnlock()
		return ErrNotFound
	}

	if item.ExpiredAt > utils.UnixTime() {
		// 这时如果callback处理比较慢的话，可能会影响性能，但目前没有更好的解决方案
		callback(item.Value, int64(len(item.Value)), item.ExpiredAt, true)
		this.locker.RUnlock()
		return nil
	}
	this.locker.RUnlock()

	_ = this.Delete(key)

	return ErrNotFound
}

// 打开缓存写入器等待写入
func (this *MemoryStorage) Open(key string, expiredAt int64) (Writer, error) {
	// 检查是否超出最大值
	if this.policy.MaxKeys > 0 && this.list.Count() > this.policy.MaxKeys {
		return nil, errors.New("write memory cache failed: too many keys in cache storage")
	}
	if this.policy.CapacityBytes() > 0 && this.policy.CapacityBytes() <= this.totalSize {
		return nil, errors.New("write memory cache failed: over memory size, real size: " + strconv.FormatInt(this.totalSize, 10) + " bytes")
	}

	// 先删除
	err := this.Delete(key)
	if err != nil {
		return nil, err
	}

	return NewMemoryWriter(this.valuesMap, key, expiredAt, this.locker), nil
}

// 删除某个键值对应的缓存
func (this *MemoryStorage) Delete(key string) error {
	hash := this.hash(key)
	this.locker.Lock()
	delete(this.valuesMap, hash)
	this.list.Remove(fmt.Sprintf("%d", hash))
	this.locker.Unlock()
	return nil
}

// 统计缓存
func (this *MemoryStorage) Stat() (*Stat, error) {
	this.locker.RLock()
	defer this.locker.RUnlock()

	return this.list.Stat(func(hash string) bool {
		return true
	}), nil
}

// 清除所有缓存
func (this *MemoryStorage) CleanAll() error {
	this.locker.Lock()
	this.valuesMap = map[uint64]*MemoryItem{}
	this.list.Reset()
	atomic.StoreInt64(&this.totalSize, 0)
	this.locker.Unlock()
	return nil
}

// 批量删除缓存
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

// 停止缓存策略
func (this *MemoryStorage) Stop() {
	this.locker.Lock()
	defer this.locker.Unlock()

	this.valuesMap = map[uint64]*MemoryItem{}
	this.list.Reset()
	if this.ticker != nil {
		this.ticker.Stop()
	}
}

// 获取当前存储的Policy
func (this *MemoryStorage) Policy() *serverconfigs.HTTPCachePolicy {
	return this.policy
}

// 将缓存添加到列表
func (this *MemoryStorage) AddToList(item *Item) {
	item.Size = item.ValueSize + int64(len(item.Key)) + 32 /** 32是我们评估的数据结构的长度 **/
	hash := fmt.Sprintf("%d", this.hash(item.Key))
	this.list.Add(hash, item)
}

// 计算Key Hash
func (this *MemoryStorage) hash(key string) uint64 {
	return xxhash.Sum64String(key)
}

// 清理任务
func (this *MemoryStorage) purgeLoop() {
	this.list.Purge(1000, func(hash string) {
		uintHash, err := strconv.ParseUint(hash, 10, 64)
		if err == nil {
			this.locker.Lock()
			delete(this.valuesMap, uintHash)
			this.locker.Unlock()
		}
	})
}
