package expires

import (
	"github.com/TeaOSLab/EdgeNode/internal/zero"
	"sync"
)

type ItemMap = map[uint64]zero.Zero

type List struct {
	expireMap map[int64]ItemMap // expires timestamp => map[id]ItemMap
	itemsMap  map[uint64]int64  // itemId => timestamp

	locker sync.Mutex

	gcCallback      func(itemId uint64)
	gcBatchCallback func(itemIds ItemMap)

	lastTimestamp int64
}

func NewList() *List {
	var list = &List{
		expireMap: map[int64]ItemMap{},
		itemsMap:  map[uint64]int64{},
	}

	SharedManager.Add(list)

	return list
}

func NewSingletonList() *List {
	var list = &List{
		expireMap: map[int64]ItemMap{},
		itemsMap:  map[uint64]int64{},
	}

	return list
}

// Add 添加条目
// 如果条目已经存在，则覆盖
func (this *List) Add(itemId uint64, expiresAt int64) {
	this.locker.Lock()
	defer this.locker.Unlock()

	if this.lastTimestamp == 0 || this.lastTimestamp > expiresAt {
		this.lastTimestamp = expiresAt
	}

	// 是否已经存在
	oldExpiresAt, ok := this.itemsMap[itemId]
	if ok {
		if oldExpiresAt == expiresAt {
			return
		}
		delete(this.expireMap, oldExpiresAt)
	}

	expireItemMap, ok := this.expireMap[expiresAt]
	if ok {
		expireItemMap[itemId] = zero.New()
	} else {
		expireItemMap = ItemMap{
			itemId: zero.New(),
		}
		this.expireMap[expiresAt] = expireItemMap
	}

	this.itemsMap[itemId] = expiresAt
}

func (this *List) Remove(itemId uint64) {
	this.locker.Lock()
	defer this.locker.Unlock()
	this.removeItem(itemId)
}

func (this *List) ExpiresAt(itemId uint64) int64 {
	this.locker.Lock()
	defer this.locker.Unlock()
	return this.itemsMap[itemId]
}

func (this *List) GC(timestamp int64) ItemMap {
	if this.lastTimestamp > timestamp+1 {
		return nil
	}
	this.locker.Lock()
	var itemMap = this.gcItems(timestamp)
	if len(itemMap) == 0 {
		this.locker.Unlock()
		return itemMap
	}
	this.locker.Unlock()

	if this.gcCallback != nil {
		for itemId := range itemMap {
			this.gcCallback(itemId)
		}
	}
	if this.gcBatchCallback != nil {
		this.gcBatchCallback(itemMap)
	}

	return itemMap
}

func (this *List) Clean() {
	this.locker.Lock()
	this.itemsMap = map[uint64]int64{}
	this.expireMap = map[int64]ItemMap{}
	this.locker.Unlock()
}

func (this *List) Count() int {
	this.locker.Lock()
	var count = len(this.itemsMap)
	this.locker.Unlock()
	return count
}

func (this *List) OnGC(callback func(itemId uint64)) *List {
	this.gcCallback = callback
	return this
}

func (this *List) OnGCBatch(callback func(itemMap ItemMap)) *List {
	this.gcBatchCallback = callback
	return this
}

func (this *List) removeItem(itemId uint64) {
	expiresAt, ok := this.itemsMap[itemId]
	if !ok {
		return
	}
	delete(this.itemsMap, itemId)

	expireItemMap, ok := this.expireMap[expiresAt]
	if ok {
		delete(expireItemMap, itemId)
		if len(expireItemMap) == 0 {
			delete(this.expireMap, expiresAt)
		}
	}
}

func (this *List) gcItems(timestamp int64) ItemMap {
	expireItemsMap, ok := this.expireMap[timestamp]
	if ok {
		for itemId := range expireItemsMap {
			delete(this.itemsMap, itemId)
		}
		delete(this.expireMap, timestamp)
	}
	return expireItemsMap
}
