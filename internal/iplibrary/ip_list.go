package iplibrary

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/iputils"
	"github.com/TeaOSLab/EdgeNode/internal/utils/expires"
	"github.com/TeaOSLab/EdgeNode/internal/utils/fasttime"
	"sort"
	"sync"
)

var GlobalBlackIPList = NewIPList()
var GlobalWhiteIPList = NewIPList()

// IPList IP名单
// TODO 对ipMap进行分区
type IPList struct {
	isDeleted bool

	itemsMap map[uint64]*IPItem // id => item

	sortedRangeItems []*IPItem
	ipMap            map[string]*IPItem // ipFrom => IPItem
	bufferItemsMap   map[uint64]*IPItem // id => IPItem

	allItemsMap map[uint64]*IPItem // id => item

	expireList *expires.List

	mu sync.RWMutex
}

func NewIPList() *IPList {
	var list = &IPList{
		itemsMap:       map[uint64]*IPItem{},
		bufferItemsMap: map[uint64]*IPItem{},
		allItemsMap:    map[uint64]*IPItem{},
		ipMap:          map[string]*IPItem{},
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

	if !IsZero(item.IPTo) {
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
func (this *IPList) Contains(ipBytes []byte) bool {
	if this.isDeleted {
		return false
	}

	this.mu.RLock()
	defer this.mu.RUnlock()

	if len(this.allItemsMap) > 0 {
		return true
	}

	var item = this.lookupIP(ipBytes)
	return item != nil
}

// ContainsExpires 判断是否包含某个IP
func (this *IPList) ContainsExpires(ipBytes []byte) (expiresAt int64, ok bool) {
	if this.isDeleted {
		return
	}

	this.mu.RLock()
	defer this.mu.RUnlock()

	if len(this.allItemsMap) > 0 {
		return 0, true
	}

	var item = this.lookupIP(ipBytes)

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
		item = this.lookupIP(iputils.ToBytes(ipString))
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

func (this *IPList) IPMap() map[string]*IPItem {
	return this.ipMap
}

func (this *IPList) ItemsMap() map[uint64]*IPItem {
	return this.itemsMap
}

func (this *IPList) AllItemsMap() map[uint64]*IPItem {
	return this.allItemsMap
}

func (this *IPList) BufferItemsMap() map[uint64]*IPItem {
	return this.bufferItemsMap
}

func (this *IPList) addItem(item *IPItem, lock bool, sortable bool) {
	if item == nil {
		return
	}

	if item.ExpiredAt > 0 && item.ExpiredAt < fasttime.Now().Unix() {
		return
	}

	var shouldSort bool

	if iputils.CompareBytes(item.IPFrom, item.IPTo) == 0 {
		item.IPTo = nil
	}

	if IsZero(item.IPFrom) && IsZero(item.IPTo) {
		if item.Type != IPItemTypeAll {
			return
		}
	} else if !IsZero(item.IPTo) {
		if iputils.CompareBytes(item.IPFrom, item.IPTo) > 0 {
			item.IPFrom, item.IPTo = item.IPTo, item.IPFrom
		} else if IsZero(item.IPFrom) {
			item.IPFrom = item.IPTo
			item.IPTo = nil
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
	if item.Type == IPItemTypeAll {
		this.allItemsMap[item.Id] = item
	} else if !IsZero(item.IPFrom) {
		if !IsZero(item.IPTo) {
			this.sortedRangeItems = append(this.sortedRangeItems, item)
			shouldSort = true
		} else {
			this.ipMap[ToHex(item.IPFrom)] = item
		}
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
			if iputils.CompareBytes(item1.IPFrom, item2.IPFrom) == 0 {
				return iputils.CompareBytes(item1.IPTo, item2.IPTo) < 0
			}
			return iputils.CompareBytes(item1.IPFrom, item2.IPFrom) < 0
		})
	}
}

// 不加锁的情况下查找Item
func (this *IPList) lookupIP(ipBytes []byte) *IPItem {
	{
		item, ok := this.ipMap[ToHex(ipBytes)]
		if ok && (item.ExpiredAt == 0 || item.ExpiredAt > fasttime.Now().Unix()) {
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
		var cmp = iputils.CompareBytes(item.IPFrom, ipBytes)
		if cmp < 0 {
			if iputils.CompareBytes(item.IPTo, ipBytes) >= 0 {
				resultIndex = i
			}
			return false
		} else if cmp == 0 {
			resultIndex = i
			return false
		}
		return true
	})

	if resultIndex < 0 || resultIndex >= count {
		return nil
	}

	var item = this.sortedRangeItems[resultIndex]
	if item.ExpiredAt == 0 || item.ExpiredAt > fasttime.Now().Unix() {
		return item
	}
	return nil
}

// 在不加锁的情况下删除某个Item
// 将会被别的方法引用，切记不能加锁
func (this *IPList) deleteItem(itemId uint64) {
	// 从buffer中删除
	delete(this.bufferItemsMap, itemId)

	// 从all items中删除
	_, ok := this.allItemsMap[itemId]
	if ok {
		delete(this.allItemsMap, itemId)
	}

	// 检查是否存在
	oldItem, existsOld := this.itemsMap[itemId]
	if !existsOld {
		return
	}

	// 从ipMap中删除
	if IsZero(oldItem.IPTo) {
		var ipHex = ToHex(oldItem.IPFrom)
		ipItem, ok := this.ipMap[ipHex]
		if ok && ipItem.Id == itemId {
			delete(this.ipMap, ipHex)
		}
	}

	delete(this.itemsMap, itemId)

	// 删除排序中的Item
	if !IsZero(oldItem.IPTo) {
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
