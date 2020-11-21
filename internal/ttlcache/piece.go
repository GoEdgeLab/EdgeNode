package ttlcache

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"sync"
	"time"
)

type Piece struct {
	m        map[uint64]*Item
	maxItems int
	locker   sync.RWMutex
}

func NewPiece(maxItems int) *Piece {
	return &Piece{m: map[uint64]*Item{}, maxItems: maxItems}
}

func (this *Piece) Add(key uint64, item *Item) () {
	this.locker.Lock()
	if len(this.m) >= this.maxItems {
		this.locker.Unlock()
		return
	}
	this.m[key] = item
	this.locker.Unlock()
}

func (this *Piece) Delete(key uint64) {
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
	this.locker.Lock()
	timestamp := time.Now().Unix()
	for k, item := range this.m {
		if item.expiredAt <= timestamp {
			delete(this.m, k)
		}
	}
	this.locker.Unlock()
}
