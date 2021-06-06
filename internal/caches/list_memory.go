package caches

import (
	"strings"
	"sync"
)

// MemoryList 内存缓存列表管理
type MemoryList struct {
	m        map[string]*Item // hash => item
	locker   sync.RWMutex
	onAdd    func(item *Item)
	onRemove func(item *Item)
}

func NewMemoryList() ListInterface {
	return &MemoryList{
		m: map[string]*Item{},
	}
}

func (this *MemoryList) Init() error {
	// 内存列表不需要初始化
	return nil
}

func (this *MemoryList) Reset() error {
	this.locker.Lock()
	this.m = map[string]*Item{}
	this.locker.Unlock()
	return nil
}

func (this *MemoryList) Add(hash string, item *Item) error {
	this.locker.Lock()

	// 先删除，为了可以正确触发统计
	oldItem, ok := this.m[hash]
	if ok {
		if this.onRemove != nil {
			this.onRemove(oldItem)
		}
	}

	// 添加
	if this.onAdd != nil {
		this.onAdd(item)
	}
	this.m[hash] = item
	this.locker.Unlock()
	return nil
}

func (this *MemoryList) Exist(hash string) (bool, error) {
	this.locker.RLock()
	defer this.locker.RUnlock()

	item, ok := this.m[hash]
	if !ok {
		return false, nil
	}

	return !item.IsExpired(), nil
}

// FindKeysWithPrefix 根据前缀进行查找
func (this *MemoryList) FindKeysWithPrefix(prefix string) (keys []string, err error) {
	this.locker.RLock()
	defer this.locker.RUnlock()

	// TODO 需要优化性能，支持千万级数据低于1s的处理速度
	for _, item := range this.m {
		if strings.HasPrefix(item.Key, prefix) {
			keys = append(keys, item.Key)
		}
	}
	return
}

func (this *MemoryList) Remove(hash string) error {
	this.locker.Lock()

	item, ok := this.m[hash]
	if ok {
		if this.onRemove != nil {
			this.onRemove(item)
		}
		delete(this.m, hash)
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
	for hash, item := range this.m {
		if count <= 0 {
			break
		}

		if item.IsExpired() {
			if this.onRemove != nil {
				this.onRemove(item)
			}
			delete(this.m, hash)
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
	for hash, item := range this.m {
		if !item.IsExpired() {
			// 检查文件是否存在、内容是否正确等
			if check != nil && check(hash) {
				result.Count++
				result.ValueSize += item.Size()
				result.Size += item.TotalSize()
			}
		}
	}
	return result, nil
}

// Count 总数量
func (this *MemoryList) Count() (int64, error) {
	this.locker.RLock()
	count := int64(len(this.m))
	this.locker.RUnlock()
	return count, nil
}

// OnAdd 添加事件
func (this *MemoryList) OnAdd(f func(item *Item)) {
	this.onAdd = f
}

// OnRemove 删除事件
func (this *MemoryList) OnRemove(f func(item *Item)) {
	this.onRemove = f
}
