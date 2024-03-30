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
// TODO 考虑将ipv6单独放入buckets
// TODO 对ipMap进行分区
type IPList struct {
	isDeleted bool

	itemsMap map[uint64]*IPItem // id => item

	sortedRangeItems []*IPItem
	ipMap            map[uint64]*IPItem // ipFrom => *IPItem
	bufferItemsMap   map[uint64]*IPItem // id => *IPItem

	allItemsMap map[uint64]*IPItem // id => item

	expireList *expires.List

	mu sync.RWMutex
}

func NewIPList() *IPList {
	var list = &IPList{
		itemsMap:       map[uint64]*IPItem{},
		bufferItemsMap: map[uint64]*IPItem{},
		allItemsMap:    map[uint64]*IPItem{},
		ipMap:          map[uint64]*IPItem{},
	}

	var expireList = expires.NewList()
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

	this.addItem(item, true, true)
}

func (this *IPList) AddDelay(item *IPItem) {
	if this.isDeleted || item == nil {
		return
	}

	if item.IPTo > 0 {
		this.mu.Lock()
		this.bufferItemsMap[item.Id] = item
		this.mu.Unlock()
	} else {
		this.addItem(item, true, true)
	}
}

func (this *IPList) Sort() {
	this.mu.Lock()
	this.sortRangeItems(false)
	this.mu.Unlock()
}

func (this *IPList) Delete(itemId uint64) {
	this.mu.Lock()
	this.deleteItem(itemId)
	this.mu.Unlock()
}

// Contains 判断是否包含某个IP
func (this *IPList) Contains(ip uint64) bool {
	if this.isDeleted {
		return false
	}

	this.mu.RLock()
	defer this.mu.RUnlock()

	if len(this.allItemsMap) > 0 {
		return true
	}

	var item = this.lookupIP(ip)

	return item != nil
}

// ContainsExpires 判断是否包含某个IP
func (this *IPList) ContainsExpires(ip uint64) (expiresAt int64, ok bool) {
	if this.isDeleted {
		return
	}

	this.mu.RLock()
	defer this.mu.RUnlock()

	if len(this.allItemsMap) > 0 {
		return 0, true
	}

	var item = this.lookupIP(ip)

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

	this.mu.RLock()
	defer this.mu.RUnlock()

	if len(this.allItemsMap) > 0 {
		for _, allItem := range this.allItemsMap {
			item = allItem
			break
		}

		if item != nil {
			found = true
			return
		}
	}
	for _, ipString := range ipStrings {
		if len(ipString) == 0 {
			continue
		}
		item = this.lookupIP(utils.IP2LongHash(ipString))
		if item != nil {
			found = true
			return
		}
	}
	return
}

func (this *IPList) SetDeleted() {
	this.isDeleted = true
}

func (this *IPList) SortedRangeItems() []*IPItem {
	return this.sortedRangeItems
}

func (this *IPList) IPMap() map[uint64]*IPItem {
	return this.ipMap
}

func (this *IPList) ItemsMap() map[uint64]*IPItem {
	return this.itemsMap
}

func (this *IPList) AllItemsMap() map[uint64]*IPItem {
	return this.allItemsMap
}

func (this *IPList) addItem(item *IPItem, lock bool, sortable bool) {
	if item == nil {
		return
	}

	if item.ExpiredAt > 0 && item.ExpiredAt < fasttime.Now().Unix() {
		return
	}

	var shouldSort bool

	if item.IPFrom == item.IPTo {
		item.IPTo = 0
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

	if lock {
		this.mu.Lock()
		defer this.mu.Unlock()
	}

	// 是否已经存在
	_, ok := this.itemsMap[item.Id]
	if ok {
		this.deleteItem(item.Id)
	}

	this.itemsMap[item.Id] = item

	// 展开
	if item.IPFrom > 0 {
		if item.IPTo > 0 {
			this.sortedRangeItems = append(this.sortedRangeItems, item)
			shouldSort = true
		} else {
			this.ipMap[item.IPFrom] = item
		}
	} else {
		this.allItemsMap[item.Id] = item
	}

	if item.ExpiredAt > 0 {
		this.expireList.Add(item.Id, item.ExpiredAt)
	}

	if shouldSort && sortable {
		this.sortRangeItems(true)
	}
}

// 对列表进行排序
func (this *IPList) sortRangeItems(force bool) {
	if len(this.bufferItemsMap) > 0 {
		for _, item := range this.bufferItemsMap {
			this.addItem(item, false, false)
		}
		this.bufferItemsMap = map[uint64]*IPItem{}
		force = true
	}

	if force {
		sort.Slice(this.sortedRangeItems, func(i, j int) bool {
			var item1 = this.sortedRangeItems[i]
			var item2 = this.sortedRangeItems[j]
			if item1.IPFrom == item2.IPFrom {
				return item1.IPTo < item2.IPTo
			}
			return item1.IPFrom < item2.IPFrom
		})
	}
}

// 不加锁的情况下查找Item
func (this *IPList) lookupIP(ip uint64) *IPItem {
	{
		item, ok := this.ipMap[ip]
		if ok {
			return item
		}
	}

	if len(this.sortedRangeItems) == 0 {
		return nil
	}

	var count = len(this.sortedRangeItems)
	var resultIndex = -1
	sort.Search(count, func(i int) bool {
		var item = this.sortedRangeItems[i]
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

	return this.sortedRangeItems[resultIndex]
}

// 在不加锁的情况下删除某个Item
// 将会被别的方法引用，切记不能加锁
func (this *IPList) deleteItem(itemId uint64) {
	// 从buffer中删除
	delete(this.bufferItemsMap, itemId)

	// 检查是否存在
	oldItem, existsOld := this.itemsMap[itemId]
	if !existsOld {
		return
	}

	// 从ipMap中删除
	if oldItem.IPTo == 0 {
		ipItem, ok := this.ipMap[oldItem.IPFrom]
		if ok && ipItem.Id == itemId {
			delete(this.ipMap, oldItem.IPFrom)
		}
	}

	delete(this.itemsMap, itemId)

	// 是否为All Item
	_, ok := this.allItemsMap[itemId]
	if ok {
		delete(this.allItemsMap, itemId)
		return
	}

	// 删除排序中的Item
	if oldItem.IPTo > 0 {
		var index = -1
		for itemIndex, item := range this.sortedRangeItems {
			if item.Id == itemId {
				index = itemIndex
				break
			}
		}
		if index >= 0 {
			copy(this.sortedRangeItems[index:], this.sortedRangeItems[index+1:])
			this.sortedRangeItems = this.sortedRangeItems[:len(this.sortedRangeItems)-1]
		}
	}
}
