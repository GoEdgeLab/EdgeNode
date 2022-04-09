package ttlcache

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/TeaOSLab/EdgeNode/internal/utils/expires"
	"github.com/iwind/TeaGo/types"
	"sync"
	"time"
)

type Piece struct {
	m           map[uint64]*Item
	expiresList *expires.List
	maxItems    int
	lastGCTime  int64

	locker sync.RWMutex
}

func NewPiece(maxItems int) *Piece {
	return &Piece{
		m:           map[uint64]*Item{},
		expiresList: expires.NewSingletonList(),
		maxItems:    maxItems,
	}
}

func (this *Piece) Add(key uint64, item *Item) (ok bool) {
	this.locker.Lock()
	if len(this.m) >= this.maxItems {
		this.locker.Unlock()
		return
	}
	this.m[key] = item
	this.locker.Unlock()

	this.expiresList.Add(key, item.expiredAt)

	return true
}

func (this *Piece) IncreaseInt64(key uint64, delta int64, expiredAt int64) (result int64) {
	this.locker.Lock()
	item, ok := this.m[key]
	if ok && item.expiredAt > time.Now().Unix() {
		result = types.Int64(item.Value) + delta
		item.Value = result
		item.expiredAt = expiredAt
		this.expiresList.Add(key, expiredAt)
	} else {
		if len(this.m) < this.maxItems {
			result = delta
			this.m[key] = &Item{
				Value:     delta,
				expiredAt: expiredAt,
			}
			this.expiresList.Add(key, expiredAt)
		}
	}
	this.locker.Unlock()

	return
}

func (this *Piece) Delete(key uint64) {
	this.expiresList.Remove(key)

	this.locker.Lock()
	delete(this.m, key)
	this.locker.Unlock()
}

func (this *Piece) Read(key uint64) (item *Item) {
	this.locker.RLock()
	item = this.m[key]
	if item != nil && item.expiredAt < utils.UnixTime() {
		item = nil
	}
	this.locker.RUnlock()

	return
}

func (this *Piece) Count() (count int) {
	this.locker.RLock()
	count = len(this.m)
	this.locker.RUnlock()
	return
}

func (this *Piece) GC() {
	var currentTime = time.Now().Unix()
	if this.lastGCTime == 0 {
		this.lastGCTime = currentTime - 3600
	}

	var min = this.lastGCTime
	var max = currentTime
	if min > max {
		// 过去的时间比现在大，则从这一秒重新开始
		min = max
	}

	for i := min; i <= max; i++ {
		var itemMap = this.expiresList.GC(i)
		if len(itemMap) > 0 {
			this.gcItemMap(itemMap)
		}
	}

	this.lastGCTime = currentTime
}

func (this *Piece) Clean() {
	this.locker.Lock()
	this.m = map[uint64]*Item{}
	this.locker.Unlock()

	this.expiresList.Clean()
}

func (this *Piece) Destroy() {
	this.locker.Lock()
	this.m = nil
	this.locker.Unlock()

	this.expiresList.Clean()
}

func (this *Piece) gcItemMap(itemMap expires.ItemMap) {
	this.locker.Lock()
	for key := range itemMap {
		delete(this.m, key)
	}
	this.locker.Unlock()
}
