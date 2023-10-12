package ttlcache

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/TeaOSLab/EdgeNode/internal/utils/fasttime"
	"runtime"
)

var SharedInt64Cache = NewBigCache[int64]()

// Cache TTL缓存
// 最大的缓存时间为30 * 86400
// Piece数据结构：
//
//	    Piece1            |  Piece2 | Piece3 | ...
//	[ Item1, Item2, ... ] |   ...
type Cache[T any] struct {
	isDestroyed bool
	pieces      []*Piece[T]
	countPieces uint64
	maxItems    int

	maxPiecesPerGC int
	gcPieceIndex   int
}

func NewBigCache[T any]() *Cache[T] {
	var delta = utils.SystemMemoryGB() / 2
	if delta <= 0 {
		delta = 1
	}
	return NewCache[T](NewMaxItemsOption(delta * 1_000_000))
}

func NewCache[T any](opt ...OptionInterface) *Cache[T] {
	var countPieces = 256
	var maxItems = 1_000_000

	var totalMemory = utils.SystemMemoryGB()
	if totalMemory < 2 {
		// 我们限制内存过小的服务能够使用的数量
		maxItems = 500_000
	} else {
		var delta = totalMemory / 4
		if delta > 0 {
			maxItems *= delta
		}
	}

	for _, option := range opt {
		if option == nil {
			continue
		}
		switch o := option.(type) {
		case *PiecesOption:
			if o.Count > 0 {
				countPieces = o.Count
			}
		case *MaxItemsOption:
			if o.Count > 0 {
				maxItems = o.Count
			}
		}
	}

	var maxPiecesPerGC = 4
	var numCPU = runtime.NumCPU() / 2
	if numCPU > maxPiecesPerGC {
		maxPiecesPerGC = numCPU
	}

	var cache = &Cache[T]{
		countPieces:    uint64(countPieces),
		maxItems:       maxItems,
		maxPiecesPerGC: maxPiecesPerGC,
	}

	for i := 0; i < countPieces; i++ {
		cache.pieces = append(cache.pieces, NewPiece[T](maxItems/countPieces))
	}

	// Add to manager
	SharedManager.Add(cache)

	return cache
}

func (this *Cache[T]) Write(key string, value T, expiredAt int64) (ok bool) {
	if this.isDestroyed {
		return
	}

	var currentTimestamp = fasttime.Now().Unix()
	if expiredAt <= currentTimestamp {
		return
	}

	var maxExpiredAt = currentTimestamp + 30*86400
	if expiredAt > maxExpiredAt {
		expiredAt = maxExpiredAt
	}
	var uint64Key = HashKey([]byte(key))
	var pieceIndex = uint64Key % this.countPieces
	return this.pieces[pieceIndex].Add(uint64Key, &Item[T]{
		Value:     value,
		expiredAt: expiredAt,
	})
}

func (this *Cache[T]) IncreaseInt64(key string, delta T, expiredAt int64, extend bool) T {
	if this.isDestroyed {
		return any(0).(T)
	}

	var currentTimestamp = fasttime.Now().Unix()
	if expiredAt <= currentTimestamp {
		return any(0).(T)
	}

	var maxExpiredAt = currentTimestamp + 30*86400
	if expiredAt > maxExpiredAt {
		expiredAt = maxExpiredAt
	}
	var uint64Key = HashKey([]byte(key))
	var pieceIndex = uint64Key % this.countPieces
	return this.pieces[pieceIndex].IncreaseInt64(uint64Key, delta, expiredAt, extend)
}

func (this *Cache[T]) Read(key string) (item *Item[T]) {
	var uint64Key = HashKey([]byte(key))
	return this.pieces[uint64Key%this.countPieces].Read(uint64Key)
}

func (this *Cache[T]) Delete(key string) {
	var uint64Key = HashKey([]byte(key))
	this.pieces[uint64Key%this.countPieces].Delete(uint64Key)
}

func (this *Cache[T]) Count() (count int) {
	for _, piece := range this.pieces {
		count += piece.Count()
	}
	return
}

func (this *Cache[T]) GC() {
	var index = this.gcPieceIndex

	for i := index; i < index+this.maxPiecesPerGC; i++ {
		if i >= int(this.countPieces) {
			break
		}
		this.pieces[i].GC()
	}

	index += this.maxPiecesPerGC
	if index >= int(this.countPieces) {
		index = 0
	}
	this.gcPieceIndex = index
}

func (this *Cache[T]) Clean() {
	for _, piece := range this.pieces {
		piece.Clean()
	}
}

func (this *Cache[T]) Destroy() {
	SharedManager.Remove(this)

	this.isDestroyed = true

	for _, piece := range this.pieces {
		piece.Destroy()
	}
}
