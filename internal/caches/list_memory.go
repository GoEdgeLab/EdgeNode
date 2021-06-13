package caches

import (
	"github.com/iwind/TeaGo/logs"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// MemoryList 内存缓存列表管理
type MemoryList struct {
	itemMaps map[string]map[string]*Item // prefix => { hash => item }
	prefixes []string
	locker   sync.RWMutex
	onAdd    func(item *Item)
	onRemove func(item *Item)

	purgeIndex int
}

func NewMemoryList() ListInterface {
	return &MemoryList{
		itemMaps: map[string]map[string]*Item{},
	}
}

func (this *MemoryList) Init() error {
	this.prefixes = []string{"000"}
	for i := 100; i <= 999; i++ {
		this.prefixes = append(this.prefixes, strconv.Itoa(i))
	}

	for _, prefix := range this.prefixes {
		this.itemMaps[prefix] = map[string]*Item{}
	}

	return nil
}

func (this *MemoryList) Reset() error {
	this.locker.Lock()
	for key := range this.itemMaps {
		this.itemMaps[key] = map[string]*Item{}
	}
	this.locker.Unlock()
	return nil
}

func (this *MemoryList) Add(hash string, item *Item) error {
	this.locker.Lock()

	prefix := this.prefix(hash)
	itemMap, ok := this.itemMaps[prefix]
	if !ok {
		itemMap = map[string]*Item{}
		this.itemMaps[prefix] = itemMap
	}

	// 先删除，为了可以正确触发统计
	oldItem, ok := itemMap[hash]
	if ok {
		if this.onRemove != nil {
			this.onRemove(oldItem)
		}
	}

	// 添加
	if this.onAdd != nil {
		this.onAdd(item)
	}

	itemMap[hash] = item
	this.locker.Unlock()
	return nil
}

func (this *MemoryList) Exist(hash string) (bool, error) {
	this.locker.RLock()
	defer this.locker.RUnlock()

	prefix := this.prefix(hash)
	itemMap, ok := this.itemMaps[prefix]
	if !ok {
		return false, nil
	}
	item, ok := itemMap[hash]
	if !ok {
		return false, nil
	}

	return !item.IsExpired(), nil
}

// CleanPrefix 根据前缀进行清除
func (this *MemoryList) CleanPrefix(prefix string) error {
	this.locker.RLock()
	defer this.locker.RUnlock()

	// TODO 需要优化性能，支持千万级数据低于1s的处理速度
	for _, itemMap := range this.itemMaps {
		for _, item := range itemMap {
			if strings.HasPrefix(item.Key, prefix) {
				item.ExpiredAt = 0
			}
		}
	}
	return nil
}

func (this *MemoryList) Remove(hash string) error {
	this.locker.Lock()

	itemMap, ok := this.itemMaps[this.prefix(hash)]
	if !ok {
		this.locker.Unlock()
		return nil
	}

	item, ok := itemMap[hash]
	if ok {
		if this.onRemove != nil {
			this.onRemove(item)
		}
		delete(itemMap, hash)
	}

	this.locker.Unlock()
	return nil
}

// Purge 清理过期的缓存
// count 每次遍历的最大数量，控制此数字可以保证每次清理的时候不用花太多时间
// callback 每次发现过期key的调用
func (this *MemoryList) Purge(count int, callback func(hash string) error) error {
	this.locker.Lock()
	deletedHashList := []string{}

	if this.purgeIndex >= len(this.prefixes) {
		this.purgeIndex = 0
	}
	prefix := this.prefixes[this.purgeIndex]

	this.purgeIndex++

	itemMap, ok := this.itemMaps[prefix]
	if !ok {
		this.locker.Unlock()
		return nil
	}
	for hash, item := range itemMap {
		if count <= 0 {
			break
		}

		if item.IsExpired() {
			if this.onRemove != nil {
				this.onRemove(item)
			}
			delete(itemMap, hash)
			deletedHashList = append(deletedHashList, hash)
		}

		count--
	}
	this.locker.Unlock()

	// 执行外部操作
	for _, hash := range deletedHashList {
		if callback != nil {
			err := callback(hash)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (this *MemoryList) CleanAll() error {
	return this.Reset()
}

func (this *MemoryList) Stat(check func(hash string) bool) (*Stat, error) {
	this.locker.RLock()
	defer this.locker.RUnlock()

	result := &Stat{
		Count: 0,
		Size:  0,
	}
	for _, itemMap := range this.itemMaps {
		for hash, item := range itemMap {
			if !item.IsExpired() {
				// 检查文件是否存在、内容是否正确等
				if check != nil && check(hash) {
					result.Count++
					result.ValueSize += item.Size()
					result.Size += item.TotalSize()
				}
			}
		}
	}
	return result, nil
}

// Count 总数量
func (this *MemoryList) Count() (int64, error) {
	this.locker.RLock()
	var count = 0
	for _, itemMap := range this.itemMaps {
		count += len(itemMap)
	}
	this.locker.RUnlock()
	return int64(count), nil
}

// OnAdd 添加事件
func (this *MemoryList) OnAdd(f func(item *Item)) {
	this.onAdd = f
}

// OnRemove 删除事件
func (this *MemoryList) OnRemove(f func(item *Item)) {
	this.onRemove = f
}

func (this *MemoryList) Close() error {
	return nil
}

func (this *MemoryList) print(t *testing.T) {
	this.locker.Lock()
	for _, itemMap := range this.itemMaps {
		if len(itemMap) > 0 {
			logs.PrintAsJSON(itemMap, t)
		}
	}
	this.locker.Unlock()
}

func (this *MemoryList) prefix(hash string) string {
	var prefix string
	if len(hash) > 3 {
		prefix = hash[:3]
	} else {
		prefix = hash
	}
	_, ok := this.itemMaps[prefix]
	if !ok {
		prefix = "000"
	}
	return prefix
}
