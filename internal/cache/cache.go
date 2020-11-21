package cache

import (
	"time"
)

// TTL缓存
// 最大的缓存时间为30 * 86400
// Piece数据结构：
//      Piece1          |  Piece2 | Piece3 | ...
//  [ Item1, Item2, ... |   ...
// KeyMap列表数据结构
// { timestamp1 => [key1, key2, ...] }, ...
type Cache struct {
	pieces      []*Piece
	countPieces uint64

	gcPieceIndex int
}

func NewCache(opt ...OptionInterface) *Cache {
	countPieces := 128
	for _, option := range opt {
		if option == nil {
			continue
		}
		switch o := option.(type) {
		case *PiecesOption:
			if o.Count > 0 {
				countPieces = o.Count
			}
		}
	}

	cache := &Cache{
		countPieces: uint64(countPieces),
	}

	for i := 0; i < countPieces; i++ {
		cache.pieces = append(cache.pieces, NewPiece())
	}

	// start timer
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		for range ticker.C {
			cache.GC()
		}
	}()

	return cache
}

func (this *Cache) Add(key string, value interface{}, expiredAt int64) {
	currentTimestamp := time.Now().Unix()
	if expiredAt <= currentTimestamp {
		return
	}

	maxExpiredAt := currentTimestamp + 30*86400
	if expiredAt > maxExpiredAt {
		expiredAt = maxExpiredAt
	}
	uint64Key := HashKey([]byte(key))
	pieceIndex := uint64Key % this.countPieces
	this.pieces[pieceIndex].Add(uint64Key, &Item{
		value:     value,
		expiredAt: expiredAt,
	})
}

func (this *Cache) Read(key string) (value *Item) {
	uint64Key := HashKey([]byte(key))
	return this.pieces[uint64Key%this.countPieces].Read(uint64Key)
}

func (this *Cache) readIntKey(key uint64) (value *Item) {
	return this.pieces[key%this.countPieces].Read(key)
}

func (this *Cache) Delete(key string) {
	uint64Key := HashKey([]byte(key))
	this.pieces[uint64Key%this.countPieces].Delete(uint64Key)
}

func (this *Cache) deleteIntKey(key uint64) {
	this.pieces[key%this.countPieces].Delete(key)
}

func (this *Cache) Count() (count int) {
	for _, piece := range this.pieces {
		count += piece.Count()
	}
	return
}

func (this *Cache) GC() {
	this.pieces[this.gcPieceIndex].GC()
	newIndex := this.gcPieceIndex + 1
	if newIndex >= int(this.countPieces) {
		newIndex = 0
	}
	this.gcPieceIndex = newIndex
}
