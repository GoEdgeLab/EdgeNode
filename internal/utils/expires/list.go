package expires

import (
	"sync"
	"time"
)

type ItemMap = map[int64]bool

type List struct {
	expireMap map[int64]ItemMap // expires timestamp => map[id]bool
	itemsMap  map[int64]int64   // itemId => timestamp

	locker sync.Mutex
	ticker *time.Ticker
}

func NewList() *List {
	return &List{
		expireMap: map[int64]ItemMap{},
		itemsMap:  map[int64]int64{},
	}
}

func (this *List) Add(itemId int64, expiresAt int64) {
	this.locker.Lock()
	defer this.locker.Unlock()

	// 是否已经存在
	_, ok := this.itemsMap[itemId]
	if ok {
		this.removeItem(itemId)
	}

	expireItemMap, ok := this.expireMap[expiresAt]
	if ok {
		expireItemMap[itemId] = true
	} else {
		expireItemMap = ItemMap{
			itemId: true,
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
	itemMap := this.gcItems(timestamp)
	this.locker.Unlock()

	for itemId := range itemMap {
		callback(itemId)
	}
}

func (this *List) StartGC(callback func(itemId int64)) {
	this.ticker = time.NewTicker(1 * time.Second)
	lastTimestamp := int64(0)
	for range this.ticker.C {
		timestamp := time.Now().Unix()
		if lastTimestamp == 0 {
			lastTimestamp = timestamp - 3600
		}

		if timestamp >= lastTimestamp {
			for i := lastTimestamp; i <= timestamp; i++ {
				this.GC(i, callback)
			}
		} else {
			for i := timestamp; i <= lastTimestamp; i++ {
				this.GC(i, callback)
			}
		}

		// 这样做是为了防止系统时钟突变
		lastTimestamp = timestamp
	}
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
