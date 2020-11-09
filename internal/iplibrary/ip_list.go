package iplibrary

import (
	"sync"
)

// IP名单
type IPList struct {
	itemsMap map[int64]*IPItem // id => item

	locker sync.RWMutex
}

func NewIPList() *IPList {
	return &IPList{
		itemsMap: map[int64]*IPItem{},
	}
}

func (this *IPList) Add(item *IPItem) {
	this.locker.Lock()
	this.itemsMap[item.Id] = item
	this.locker.Unlock()
}

func (this *IPList) Delete(itemId int64) {
	this.locker.Lock()
	delete(this.itemsMap, itemId)
	this.locker.Unlock()
}

// 判断是否包含某个IP
func (this *IPList) Contains(ip uint32) bool {
	// TODO 优化查询速度，可能需要把items分成两组，一组是单个的，一组是按照范围的，按照范围的再进行二分法查找
	this.locker.RLock()
	for _, item := range this.itemsMap {
		if item.Contains(ip) {
			this.locker.RUnlock()
			return true
		}
	}
	this.locker.RUnlock()

	return false
}
