package ttlcache

import (
	"github.com/iwind/TeaGo/rands"
	"testing"
	"time"
)

func TestPiece_Add(t *testing.T) {
	piece := NewPiece[int](10)
	piece.Add(1, &Item[int]{expiredAt: time.Now().Unix() + 3600})
	piece.Add(2, &Item[int]{})
	piece.Add(3, &Item[int]{})
	piece.Delete(3)
	for key, item := range piece.m {
		t.Log(key, item.Value)
	}
	t.Log(piece.Read(1))
}

func TestPiece_Add_Same(t *testing.T) {
	piece := NewPiece[int](10)
	piece.Add(1, &Item[int]{expiredAt: time.Now().Unix() + 3600})
	piece.Add(1, &Item[int]{expiredAt: time.Now().Unix() + 3600})
	for key, item := range piece.m {
		t.Log(key, item.Value)
	}
	t.Log(piece.Read(1))
}

func TestPiece_MaxItems(t *testing.T) {
	piece := NewPiece[int](10)
	for i := 0; i < 1000; i++ {
		piece.Add(uint64(i), &Item[int]{expiredAt: time.Now().Unix() + 3600})
	}
	t.Log(len(piece.m))
}

func TestPiece_GC(t *testing.T) {
	piece := NewPiece[int](10)
	piece.Add(1, &Item[int]{Value: 1, expiredAt: time.Now().Unix() + 1})
	piece.Add(2, &Item[int]{Value: 2, expiredAt: time.Now().Unix() + 1})
	piece.Add(3, &Item[int]{Value: 3, expiredAt: time.Now().Unix() + 1})
	t.Log("before gc ===")
	for key, item := range piece.m {
		t.Log(key, item.Value)
	}

	time.Sleep(1 * time.Second)
	piece.GC()

	t.Log("after gc ===")
	for key, item := range piece.m {
		t.Log(key, item.Value)
	}
}

func TestPiece_GC2(t *testing.T) {
	piece := NewPiece[int](10)
	for i := 0; i < 10_000; i++ {
		piece.Add(uint64(i), &Item[int]{Value: 1, expiredAt: time.Now().Unix() + int64(rands.Int(1, 10))})
	}

	time.Sleep(1 * time.Second)

	before := time.Now()
	piece.GC()
	t.Log(time.Since(before).Seconds()*1000, "ms")
	t.Log(piece.Count())
}
