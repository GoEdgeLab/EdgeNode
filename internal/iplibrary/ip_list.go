package iplibrary

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/TeaOSLab/EdgeNode/internal/utils/expires"
	"github.com/TeaOSLab/EdgeNode/internal/utils/fasttime"
	"sort"
	"sync"
)

var GlobalBlackIPList = NewIPList()
var GlobalWhiteIPList = NewIPList()

// IPList IP名单
// TODO IP名单可以分片关闭，这样让每一片的数据量减少，查询更快
type IPList struct {
	isDeleted bool

	itemsMap    map[uint64]*IPItem // id => item
	sortedItems []*IPItem
	allItemsMap map[uint64]*IPItem // id => item

	expireList *expires.List

	locker sync.RWMutex
}

func NewIPList() *IPList {
	list := &IPList{
		itemsMap:    map[uint64]*IPItem{},
		allItemsMap: map[uint64]*IPItem{},
	}

	expireList := expires.NewList()
	expireList.OnGC(func(itemId uint64) {
		list.Delete(itemId)
	})
	list.expireList = expireList
	return list
}

func (this *IPList) Add(item *IPItem) {
	if this.isDeleted {
		return
	}

	this.addItem(item, true)
}

// AddDelay 延迟添加，需要手工调用Sort()函数
func (this *IPList) AddDelay(item *IPItem) {
	if this.isDeleted {
		return
	}

	this.addItem(item, false)
}

func (this *IPList) Sort() {
	this.locker.Lock()
	this.sortItems()
	this.locker.Unlock()
}

func (this *IPList) Delete(itemId uint64) {
	this.locker.Lock()
	this.deleteItem(itemId)
	this.locker.Unlock()
}

// Contains 判断是否包含某个IP
func (this *IPList) Contains(ip uint64) bool {
	if this.isDeleted {
		return false
	}

	this.locker.RLock()
	if len(this.allItemsMap) > 0 {
		this.locker.RUnlock()
		return true
	}

	var item = this.lookupIP(ip)

	this.locker.RUnlock()

	return item != nil
}

// ContainsExpires 判断是否包含某个IP
func (this *IPList) ContainsExpires(ip uint64) (expiresAt int64, ok bool) {
	if this.isDeleted {
		return
	}

	this.locker.RLock()
	if len(this.allItemsMap) > 0 {
		this.locker.RUnlock()
		return 0, true
	}

	var item = this.lookupIP(ip)

	this.locker.RUnlock()

	if item == nil {
		return
	}

	return item.ExpiredAt, true
}

// ContainsIPStrings 是否包含一组IP中的任意一个，并返回匹配的第一个Item
func (this *IPList) ContainsIPStrings(ipStrings []string) (item *IPItem, found bool) {
	if this.isDeleted {
		return
	}

	if len(ipStrings) == 0 {
		return
	}
	this.locker.RLock()
	if len(this.allItemsMap) > 0 {
		for _, allItem := range this.allItemsMap {
			item = allItem
			break
		}

		if item != nil {
			this.locker.RUnlock()
			found = true
			return
		}
	}
	for _, ipString := range ipStrings {
		if len(ipString) == 0 {
			continue
		}
		item = this.lookupIP(utils.IP2Long(ipString))
		if item != nil {
			this.locker.RUnlock()
			found = true
			return
		}
	}
	this.locker.RUnlock()
	return
}

func (this *IPList) SetDeleted() {
	this.isDeleted = true
}

func (this *IPList) addItem(item *IPItem, sortable bool) {
	if item == nil {
		return
	}

	if item.ExpiredAt > 0 && item.ExpiredAt < fasttime.Now().Unix() {
		return
	}

	if item.IPFrom == 0 && item.IPTo == 0 {
		if item.Type != IPItemTypeAll {
			return
		}
	} else if item.IPTo > 0 {
		if item.IPFrom > item.IPTo {
			item.IPFrom, item.IPTo = item.IPTo, item.IPFrom
		} else if item.IPFrom == 0 {
			item.IPFrom = item.IPTo
			item.IPTo = 0
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
		this.sortedItems = append(this.sortedItems, item)
	} else {
		this.allItemsMap[item.Id] = item
	}

	if item.ExpiredAt > 0 {
		this.expireList.Add(item.Id, item.ExpiredAt)
	}

	if sortable {
		this.sortItems()
	}

	this.locker.Unlock()
}

// 对列表进行排序
func (this *IPList) sortItems() {
	sort.Slice(this.sortedItems, func(i, j int) bool {
		var item1 = this.sortedItems[i]
		var item2 = this.sortedItems[j]
		if item1.IPFrom == item2.IPFrom {
			return item1.IPTo < item2.IPTo
		}
		return item1.IPFrom < item2.IPFrom
	})
}

// 不加锁的情况下查找Item
func (this *IPList) lookupIP(ip uint64) *IPItem {
	if len(this.sortedItems) == 0 {
		return nil
	}

	var count = len(this.sortedItems)
	var resultIndex = -1
	sort.Search(count, func(i int) bool {
		var item = this.sortedItems[i]
		if item.IPFrom < ip {
			if item.IPTo >= ip {
				resultIndex = i
			}
			return false
		} else if item.IPFrom == ip {
			resultIndex = i
			return false
		}
		return true
	})

	if resultIndex < 0 || resultIndex >= count {
		return nil
	}

	return this.sortedItems[resultIndex]
}

// 在不加锁的情况下删除某个Item
// 将会被别的方法引用，切记不能加锁
func (this *IPList) deleteItem(itemId uint64) {
	_, ok := this.itemsMap[itemId]
	if !ok {
		return
	}

	delete(this.itemsMap, itemId)

	// 是否为All Item
	_, ok = this.allItemsMap[itemId]
	if ok {
		delete(this.allItemsMap, itemId)
		return
	}

	// 删除排序中的Item
	var index = -1
	for itemIndex, item := range this.sortedItems {
		if item.Id == itemId {
			index = itemIndex
			break
		}
	}
	if index >= 0 {
		copy(this.sortedItems[index:], this.sortedItems[index+1:])
		this.sortedItems = this.sortedItems[:len(this.sortedItems)-1]
	}
}
