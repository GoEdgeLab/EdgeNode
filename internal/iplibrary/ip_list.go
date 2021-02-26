package iplibrary

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/TeaOSLab/EdgeNode/internal/utils/expires"
	"sync"
)

// IP名单
type IPList struct {
	itemsMap   map[int64]*IPItem  // id => item
	ipMap      map[uint64][]int64 // ip => itemIds
	expireList *expires.List

	isAll bool

	locker sync.RWMutex
}

func NewIPList() *IPList {
	list := &IPList{
		itemsMap: map[int64]*IPItem{},
		ipMap:    map[uint64][]int64{},
	}

	expireList := expires.NewList()
	go func() {
		expireList.StartGC(func(itemId int64) {
			list.Delete(itemId)
		})
	}()
	list.expireList = expireList
	return list
}

func (this *IPList) Add(item *IPItem) {
	if item == nil {
		return
	}

	if item.IPFrom == 0 && item.IPTo == 0 {
		if item.Type != "all" {
			return
		}
	}

	this.locker.Lock()

	// 是否已经存在
	_, ok := this.itemsMap[item.Id]
	if ok {
		this.deleteItem(item.Id)
	}

	this.itemsMap[item.Id] = item

	// 展开
	if item.IPFrom > 0 {
		if item.IPTo == 0 {
			this.addIP(item.IPFrom, item.Id)
		} else {
			if item.IPFrom > item.IPTo {
				item.IPTo, item.IPFrom = item.IPFrom, item.IPTo
			}

			for i := item.IPFrom; i <= item.IPTo; i++ {
				// 最多不能超过65535，防止整个系统内存爆掉
				if i >= item.IPFrom+65535 {
					break
				}
				this.addIP(i, item.Id)
			}
		}
	} else if item.IPTo > 0 {
		this.addIP(item.IPTo, item.Id)
	} else {
		this.addIP(0, item.Id)

		// 更新isAll
		this.isAll = true
	}

	if item.ExpiredAt > 0 {
		this.expireList.Add(item.Id, item.ExpiredAt)
	}

	this.locker.Unlock()
}

func (this *IPList) Delete(itemId int64) {
	this.locker.Lock()
	defer this.locker.Unlock()
	this.deleteItem(itemId)

	// 更新isAll
	this.isAll = len(this.ipMap[0]) > 0
}

// 判断是否包含某个IP
func (this *IPList) Contains(ip uint64) bool {
	this.locker.RLock()
	if this.isAll {
		this.locker.RUnlock()
		return true
	}
	_, ok := this.ipMap[ip]
	this.locker.RUnlock()

	return ok
}

// 是否包含一组IP
func (this *IPList) ContainsIPStrings(ipStrings []string) (found bool, item *IPItem) {
	if len(ipStrings) == 0 {
		return
	}
	this.locker.RLock()
	if this.isAll {
		itemIds := this.ipMap[0]
		if len(itemIds) > 0 {
			itemId := itemIds[0]
			item = this.itemsMap[itemId]
		}

		this.locker.RUnlock()
		found = true
		return
	}
	for _, ipString := range ipStrings {
		if len(ipString) == 0 {
			continue
		}
		itemIds, ok := this.ipMap[utils.IP2Long(ipString)]
		if ok {
			if len(itemIds) > 0 {
				itemId := itemIds[0]
				item = this.itemsMap[itemId]
			}

			this.locker.RUnlock()
			found = true
			return
		}
	}
	this.locker.RUnlock()
	return
}

// 在不加锁的情况下删除某个Item
// 将会被别的方法引用，切记不能加锁
func (this *IPList) deleteItem(itemId int64) {
	item, ok := this.itemsMap[itemId]
	if !ok {
		return
	}

	delete(this.itemsMap, itemId)

	// 展开
	if item.IPFrom > 0 {
		if item.IPTo == 0 {
			this.deleteIP(item.IPFrom, item.Id)
		} else {
			if item.IPFrom > item.IPTo {
				item.IPTo, item.IPFrom = item.IPFrom, item.IPTo
			}

			for i := item.IPFrom; i <= item.IPTo; i++ {
				// 最多不能超过65535，防止整个系统内存爆掉
				if i >= item.IPFrom+65535 {
					break
				}
				this.deleteIP(i, item.Id)
			}
		}
	} else if item.IPTo > 0 {
		this.deleteIP(item.IPTo, item.Id)
	} else {
		this.deleteIP(0, item.Id)
	}
}

// 添加单个IP
func (this *IPList) addIP(ip uint64, itemId int64) {
	itemIds, ok := this.ipMap[ip]
	if ok {
		itemIds = append(itemIds, itemId)
	} else {
		itemIds = []int64{itemId}
	}
	this.ipMap[ip] = itemIds
}

// 删除单个IP
func (this *IPList) deleteIP(ip uint64, itemId int64) {
	itemIds, ok := this.ipMap[ip]
	if !ok {
		return
	}
	newItemIds := []int64{}
	for _, oldItemId := range itemIds {
		if oldItemId == itemId {
			continue
		}
		newItemIds = append(newItemIds, oldItemId)
	}
	if len(newItemIds) > 0 {
		this.ipMap[ip] = newItemIds
	} else {
		delete(this.ipMap, ip)
	}
}
