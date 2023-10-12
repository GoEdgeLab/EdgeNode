package ttlcache

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils/expires"
	"github.com/TeaOSLab/EdgeNode/internal/utils/fasttime"
	"sync"
)

type Piece[T any] struct {
	m           map[uint64]*Item[T]
	expiresList *expires.List
	maxItems    int
	lastGCTime  int64

	locker sync.RWMutex
}

func NewPiece[T any](maxItems int) *Piece[T] {
	return &Piece[T]{
		m:           map[uint64]*Item[T]{},
		expiresList: expires.NewSingletonList(),
		maxItems:    maxItems,
	}
}

func (this *Piece[T]) Add(key uint64, item *Item[T]) (ok bool) {
	this.locker.RLock()
	if this.maxItems > 0 && len(this.m) >= this.maxItems {
		this.locker.RUnlock()
		return
	}
	this.locker.RUnlock()

	this.locker.Lock()
	oldItem, exists := this.m[key]
	if exists && oldItem.expiredAt == item.expiredAt {
		this.locker.Unlock()
		return true
	}
	this.m[key] = item
	this.locker.Unlock()

	this.expiresList.Add(key, item.expiredAt)

	return true
}

func (this *Piece[T]) IncreaseInt64(key uint64, delta T, expiredAt int64, extend bool) (result T) {
	this.locker.Lock()
	item, ok := this.m[key]
	if ok && item.expiredAt > fasttime.Now().Unix() {
		int64Value, isInt64 := any(item.Value).(int64)
		if isInt64 {
			result = any(int64Value + any(delta).(int64)).(T)
		}
		item.Value = result
		if extend {
			item.expiredAt = expiredAt
		}
		this.expiresList.Add(key, expiredAt)
	} else {
		if len(this.m) < this.maxItems {
			result = delta
			this.m[key] = &Item[T]{
				Value:     delta,
				expiredAt: expiredAt,
			}
			this.expiresList.Add(key, expiredAt)
		}
	}
	this.locker.Unlock()

	return
}

func (this *Piece[T]) Delete(key uint64) {
	this.expiresList.Remove(key)

	this.locker.Lock()
	delete(this.m, key)
	this.locker.Unlock()
}

func (this *Piece[T]) Read(key uint64) (item *Item[T]) {
	this.locker.RLock()
	item = this.m[key]
	if item != nil && item.expiredAt < fasttime.Now().Unix() {
		item = nil
	}
	this.locker.RUnlock()

	return
}

func (this *Piece[T]) Count() (count int) {
	this.locker.RLock()
	count = len(this.m)
	this.locker.RUnlock()
	return
}

func (this *Piece[T]) GC() {
	var currentTime = fasttime.Now().Unix()
	if this.lastGCTime == 0 {
		this.lastGCTime = currentTime - 3600
	}

	var minTime = this.lastGCTime
	var maxTime = currentTime
	if minTime > maxTime {
		// 过去的时间比现在大，则从这一秒重新开始
		minTime = maxTime
	}

	for i := minTime; i <= maxTime; i++ {
		var itemMap = this.expiresList.GC(i)
		if len(itemMap) > 0 {
			this.gcItemMap(itemMap)
		}
	}

	this.lastGCTime = currentTime
}

func (this *Piece[T]) Clean() {
	this.locker.Lock()
	this.m = map[uint64]*Item[T]{}
	this.locker.Unlock()

	this.expiresList.Clean()
}

func (this *Piece[T]) Destroy() {
	this.locker.Lock()
	this.m = nil
	this.locker.Unlock()

	this.expiresList.Clean()
}

func (this *Piece[T]) gcItemMap(itemMap expires.ItemMap) {
	this.locker.Lock()
	for key := range itemMap {
		delete(this.m, key)
	}
	this.locker.Unlock()
}
