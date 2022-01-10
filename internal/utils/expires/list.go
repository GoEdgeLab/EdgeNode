package expires

import (
	"github.com/TeaOSLab/EdgeNode/internal/zero"
	"sync"
)

type ItemMap = map[int64]zero.Zero

type List struct {
	expireMap map[int64]ItemMap // expires timestamp => map[id]ItemMap
	itemsMap  map[int64]int64   // itemId => timestamp

	locker sync.Mutex

	gcCallback func(itemId int64)
}

func NewList() *List {
	var list = &List{
		expireMap: map[int64]ItemMap{},
		itemsMap:  map[int64]int64{},
	}

	SharedManager.Add(list)

	return list
}

// Add 添加条目
// 如果条目已经存在，则覆盖
func (this *List) Add(itemId int64, expiresAt int64) {
	this.locker.Lock()
	defer this.locker.Unlock()

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

func (this *List) Remove(itemId int64) {
	this.locker.Lock()
	defer this.locker.Unlock()
	this.removeItem(itemId)
}

func (this *List) GC(timestamp int64, callback func(itemId int64)) {
	this.locker.Lock()
	var itemMap = this.gcItems(timestamp)
	if len(itemMap) == 0 {
		this.locker.Unlock()
		return
	}
	this.locker.Unlock()

	if callback != nil {
		for itemId := range itemMap {
			callback(itemId)
		}
	}
}

func (this *List) OnGC(callback func(itemId int64)) *List {
	this.gcCallback = callback
	return this
}

func (this *List) removeItem(itemId int64) {
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
