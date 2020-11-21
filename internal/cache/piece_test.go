package cache

import (
	"github.com/iwind/TeaGo/rands"
	"testing"
	"time"
)

func TestPiece_Add(t *testing.T) {
	piece := NewPiece()
	piece.Add(1, &Item{expiredAt: time.Now().Unix() + 3600})
	piece.Add(2, &Item{})
	piece.Add(3, &Item{})
	piece.Delete(3)
	for key, item := range piece.m {
		t.Log(key, item.value)
	}
	t.Log(piece.Read(1))
}

func TestPiece_GC(t *testing.T) {
	piece := NewPiece()
	piece.Add(1, &Item{value: 1, expiredAt: time.Now().Unix() + 1})
	piece.Add(2, &Item{value: 2, expiredAt: time.Now().Unix() + 1})
	piece.Add(3, &Item{value: 3, expiredAt: time.Now().Unix() + 1})
	t.Log("before gc ===")
	for key, item := range piece.m {
		t.Log(key, item.value)
	}

	time.Sleep(1 * time.Second)
	piece.GC()

	t.Log("after gc ===")
	for key, item := range piece.m {
		t.Log(key, item.value)
	}
}

func TestPiece_GC2(t *testing.T) {
	piece := NewPiece()
	for i := 0; i < 10_000; i++ {
		piece.Add(uint64(i), &Item{value: 1, expiredAt: time.Now().Unix() + int64(rands.Int(1, 10))})
	}

	time.Sleep(1 * time.Second)

	before := time.Now()
	piece.GC()
	t.Log(time.Since(before).Seconds()*1000, "ms")
	t.Log(piece.Count())
}
