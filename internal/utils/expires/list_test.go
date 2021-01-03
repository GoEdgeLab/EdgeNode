package expires

import (
	"github.com/iwind/TeaGo/logs"
	timeutil "github.com/iwind/TeaGo/utils/time"
	"math"
	"testing"
	"time"
)

func TestList_Add(t *testing.T) {
	list := NewList()
	list.Add(1, time.Now().Unix())
	t.Log("===BEFORE===")
	logs.PrintAsJSON(list.expireMap, t)
	logs.PrintAsJSON(list.itemsMap, t)

	list.Add(1, time.Now().Unix()+1)
	list.Add(2, time.Now().Unix()+1)
	list.Add(3, time.Now().Unix()+2)
	t.Log("===AFTER===")
	logs.PrintAsJSON(list.expireMap, t)
	logs.PrintAsJSON(list.itemsMap, t)
}

func TestList_Add_Overwrite(t *testing.T) {
	list := NewList()
	list.Add(1, time.Now().Unix()+1)
	list.Add(1, time.Now().Unix()+1)
	list.Add(1, time.Now().Unix()+2)
	logs.PrintAsJSON(list.expireMap, t)
	logs.PrintAsJSON(list.itemsMap, t)
}

func TestList_Remove(t *testing.T) {
	list := NewList()
	list.Add(1, time.Now().Unix()+1)
	list.Remove(1)
	logs.PrintAsJSON(list.expireMap, t)
	logs.PrintAsJSON(list.itemsMap, t)
}

func TestList_GC(t *testing.T) {
	list := NewList()
	list.Add(1, time.Now().Unix()+1)
	list.Add(2, time.Now().Unix()+1)
	list.Add(3, time.Now().Unix()+2)
	list.GC(time.Now().Unix()+2, func(itemId int64) {
		t.Log("gc:", itemId)
	})
	logs.PrintAsJSON(list.expireMap, t)
	logs.PrintAsJSON(list.itemsMap, t)
}

func TestList_Start_GC(t *testing.T) {
	list := NewList()
	list.Add(1, time.Now().Unix()+1)
	list.Add(2, time.Now().Unix()+1)
	list.Add(3, time.Now().Unix()+2)
	list.Add(4, time.Now().Unix()+5)

	go func() {
		list.StartGC(func(itemId int64) {
			t.Log("gc:", itemId, timeutil.Format("H:i:s"))
			time.Sleep(2 * time.Second)
		})
	}()

	time.Sleep(10 * time.Second)
}

func TestList_ManyItems(t *testing.T) {
	list := NewList()
	for i := 0; i < 100_000; i++ {
		list.Add(int64(i), time.Now().Unix()+1)
	}

	now := time.Now()
	count := 0
	list.GC(time.Now().Unix()+1, func(itemId int64) {
		count++
	})
	t.Log("gc", count, "items")
	t.Log(time.Since(now).Seconds()*1000, "ms")
}

func TestList_Map_Performance(t *testing.T) {
	t.Log("max uint32", math.MaxUint32)

	{
		m := map[int64]int64{}
		for i := 0; i < 1_000_000; i++ {
			m[int64(i)] = time.Now().Unix()
		}

		now := time.Now()
		for i := 0; i < 100_000; i++ {
			delete(m, int64(i))
		}
		t.Log(time.Since(now).Seconds()*1000, "ms")
	}

	{
		m := map[uint32]int64{}
		for i := 0; i < 1_000_000; i++ {
			m[uint32(i)] = time.Now().Unix()
		}

		now := time.Now()
		for i := 0; i < 100_000; i++ {
			delete(m, uint32(i))
		}
		t.Log(time.Since(now).Seconds()*1000, "ms")
	}
}
