package caches

import "sync"

// 缓存列表管理
type List struct {
	m        map[string]*Item // hash => item
	locker   sync.RWMutex
	onAdd    func(item *Item)
	onRemove func(item *Item)
}

func NewList() *List {
	return &List{
		m: map[string]*Item{},
	}
}

func (this *List) Reset() {
	this.locker.Lock()
	this.m = map[string]*Item{}
	this.locker.Unlock()
}

func (this *List) Add(hash string, item *Item) {
	this.locker.Lock()
	if this.onAdd != nil {
		this.onAdd(item)
	}
	this.m[hash] = item
	this.locker.Unlock()
}

func (this *List) Exist(hash string) bool {
	this.locker.RLock()
	defer this.locker.RUnlock()

	item, ok := this.m[hash]
	if !ok {
		return false
	}

	return !item.IsExpired()
}

func (this *List) Remove(hash string) {
	this.locker.Lock()

	item, ok := this.m[hash]
	if ok {
		if this.onRemove != nil {
			this.onRemove(item)
		}
		delete(this.m, hash)
	}

	this.locker.Unlock()
}

// 清理过期的缓存
// count 每次遍历的最大数量，控制此数字可以保证每次清理的时候不用花太多时间
// callback 每次发现过期key的调用
func (this *List) Purge(count int, callback func(hash string)) {
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
			callback(hash)
		}
	}
}

func (this *List) Stat(check func(hash string) bool) *Stat {
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
				result.ValueSize += item.ValueSize
				result.Size += item.Size
			}
		}
	}
	return result
}

// 总数量
func (this *List) Count() int64 {
	this.locker.RLock()
	count := int64(len(this.m))
	this.locker.RUnlock()
	return count
}

// 添加事件
func (this *List) OnAdd(f func(item *Item)) {
	this.onAdd = f
}

// 删除事件
func (this *List) OnRemove(f func(item *Item)) {
	this.onRemove = f
}
