package cache

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"sync"
	"time"
)

type Piece struct {
	m      map[uint64]*Item
	locker sync.RWMutex
}

func NewPiece() *Piece {
	return &Piece{m: map[uint64]*Item{}}
}

func (this *Piece) Add(key uint64, item *Item) () {
	this.locker.Lock()
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
