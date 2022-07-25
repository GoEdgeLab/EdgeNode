package iplibrary

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/iwind/TeaGo/assert"
	"github.com/iwind/TeaGo/logs"
	"github.com/iwind/TeaGo/rands"
	"runtime"
	"runtime/debug"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestIPList_Add_Empty(t *testing.T) {
	ipList := NewIPList()
	ipList.Add(&IPItem{
		Id: 1,
	})
	logs.PrintAsJSON(ipList.itemsMap, t)
	logs.PrintAsJSON(ipList.allItemsMap, t)
}

func TestIPList_Add_One(t *testing.T) {
	ipList := NewIPList()
	ipList.Add(&IPItem{
		Id:     1,
		IPFrom: utils.IP2Long("192.168.1.1"),
	})
	ipList.Add(&IPItem{
		Id:   2,
		IPTo: utils.IP2Long("192.168.1.2"),
	})
	ipList.Add(&IPItem{
		Id:     3,
		IPFrom: utils.IP2Long("192.168.0.2"),
	})
	ipList.Add(&IPItem{
		Id:     4,
		IPFrom: utils.IP2Long("192.168.0.2"),
		IPTo:   utils.IP2Long("192.168.0.1"),
	})
	ipList.Add(&IPItem{
		Id:     5,
		IPFrom: utils.IP2Long("2001:db8:0:1::101"),
	})
	ipList.Add(&IPItem{
		Id:     6,
		IPFrom: 0,
		Type:   "all",
	})
	t.Log("===items===")
	logs.PrintAsJSON(ipList.itemsMap, t)

	t.Log("===sorted items===")
	logs.PrintAsJSON(ipList.sortedItems, t)

	t.Log("===all items===")
	logs.PrintAsJSON(ipList.allItemsMap, t) // ip => items
}

func TestIPList_Update(t *testing.T) {
	ipList := NewIPList()
	ipList.Add(&IPItem{
		Id:     1,
		IPFrom: utils.IP2Long("192.168.1.1"),
	})
	/**ipList.Add(&IPItem{
		Id:     2,
		IPFrom: utils.IP2Long("192.168.1.1"),
	})**/
	ipList.Add(&IPItem{
		Id:   1,
		IPTo: utils.IP2Long("192.168.1.2"),
	})
	logs.PrintAsJSON(ipList.itemsMap, t)
	logs.PrintAsJSON(ipList.sortedItems, t)
}

func TestIPList_Update_AllItems(t *testing.T) {
	ipList := NewIPList()
	ipList.Add(&IPItem{
		Id:     1,
		Type:   IPItemTypeAll,
		IPFrom: 0,
	})
	ipList.Add(&IPItem{
		Id:   1,
		IPTo: 0,
	})
	t.Log("===items map===")
	logs.PrintAsJSON(ipList.itemsMap, t)
	t.Log("===all items map===")
	logs.PrintAsJSON(ipList.allItemsMap, t)
}

func TestIPList_Add_Range(t *testing.T) {
	ipList := NewIPList()
	ipList.Add(&IPItem{
		Id:     1,
		IPFrom: utils.IP2Long("192.168.1.1"),
		IPTo:   utils.IP2Long("192.168.2.1"),
	})
	ipList.Add(&IPItem{
		Id:   2,
		IPTo: utils.IP2Long("192.168.1.2"),
	})
	t.Log(len(ipList.itemsMap), "ips")
	logs.PrintAsJSON(ipList.itemsMap, t)
	logs.PrintAsJSON(ipList.allItemsMap, t)
}

func TestIPList_Add_Overflow(t *testing.T) {
	a := assert.NewAssertion(t)

	ipList := NewIPList()
	ipList.Add(&IPItem{
		Id:     1,
		IPFrom: utils.IP2Long("192.168.1.1"),
		IPTo:   utils.IP2Long("192.169.255.1"),
	})
	t.Log(len(ipList.itemsMap), "ips")
	a.IsTrue(len(ipList.itemsMap) <= 65535)
}

func TestNewIPList_Memory(t *testing.T) {
	list := NewIPList()

	for i := 0; i < 200_0000; i++ {
		list.Add(&IPItem{
			IPFrom:    1,
			IPTo:      2,
			ExpiredAt: time.Now().Unix(),
		})
	}

	t.Log("ok")
}

func TestIPList_Contains(t *testing.T) {
	var a = assert.NewAssertion(t)

	list := NewIPList()
	for i := 0; i < 255; i++ {
		list.AddDelay(&IPItem{
			Id:        uint64(i),
			IPFrom:    utils.IP2Long(strconv.Itoa(i) + ".168.0.1"),
			IPTo:      utils.IP2Long(strconv.Itoa(i) + ".168.255.1"),
			ExpiredAt: 0,
		})
	}
	for i := 0; i < 255; i++ {
		list.AddDelay(&IPItem{
			Id:     uint64(1000 + i),
			IPFrom: utils.IP2Long("192.167.2." + strconv.Itoa(i)),
		})
	}
	list.Sort()
	t.Log(len(list.itemsMap), "ip")

	before := time.Now()
	a.IsTrue(list.Contains(utils.IP2Long("192.168.1.100")))
	a.IsTrue(list.Contains(utils.IP2Long("192.168.2.100")))
	a.IsFalse(list.Contains(utils.IP2Long("192.169.3.100")))
	a.IsFalse(list.Contains(utils.IP2Long("192.167.3.100")))
	a.IsTrue(list.Contains(utils.IP2Long("192.167.2.100")))
	t.Log(time.Since(before).Seconds()*1000, "ms")
}

func TestIPList_Contains_Many(t *testing.T) {
	list := NewIPList()
	for i := 0; i < 1_000_000; i++ {
		list.AddDelay(&IPItem{
			Id:        uint64(i),
			IPFrom:    utils.IP2Long(strconv.Itoa(rands.Int(0, 255)) + "." + strconv.Itoa(rands.Int(0, 255)) + "." + strconv.Itoa(rands.Int(0, 255)) + "." + strconv.Itoa(rands.Int(0, 255))),
			IPTo:      utils.IP2Long(strconv.Itoa(rands.Int(0, 255)) + "." + strconv.Itoa(rands.Int(0, 255)) + "." + strconv.Itoa(rands.Int(0, 255)) + "." + strconv.Itoa(rands.Int(0, 255))),
			ExpiredAt: 0,
		})
	}
	list.Sort()
	t.Log(len(list.itemsMap), "ip")

	before := time.Now()
	_ = list.Contains(utils.IP2Long("192.168.1.100"))
	t.Log(time.Since(before).Seconds()*1000, "ms")
}

func TestIPList_ContainsAll(t *testing.T) {
	list := NewIPList()
	list.Add(&IPItem{
		Id:     1,
		Type:   "all",
		IPFrom: 0,
	})
	b := list.Contains(utils.IP2Long("192.168.1.1"))
	if b {
		t.Log(b)
	} else {
		t.Fatal("'b' should be true")
	}

	list.Delete(1)

	b = list.Contains(utils.IP2Long("192.168.1.1"))
	if !b {
		t.Log(b)
	} else {
		t.Fatal("'b' should be false")
	}

}

func TestIPList_ContainsIPStrings(t *testing.T) {
	var a = assert.NewAssertion(t)

	list := NewIPList()
	for i := 0; i < 255; i++ {
		list.Add(&IPItem{
			Id:        uint64(i),
			IPFrom:    utils.IP2Long(strconv.Itoa(i) + ".168.0.1"),
			IPTo:      utils.IP2Long(strconv.Itoa(i) + ".168.255.1"),
			ExpiredAt: 0,
		})
	}
	t.Log(len(list.itemsMap), "ip")

	{
		item, ok := list.ContainsIPStrings([]string{"192.168.1.100"})
		t.Log("item:", item)
		a.IsTrue(ok)
	}
	{
		item, ok := list.ContainsIPStrings([]string{"192.167.1.100"})
		t.Log("item:", item)
		a.IsFalse(ok)
	}
}

func TestIPList_Delete(t *testing.T) {
	list := NewIPList()
	list.Add(&IPItem{
		Id:        1,
		IPFrom:    utils.IP2Long("192.168.0.1"),
		ExpiredAt: 0,
	})
	list.Add(&IPItem{
		Id:        2,
		IPFrom:    utils.IP2Long("192.168.0.1"),
		ExpiredAt: 0,
	})
	t.Log("===BEFORE===")
	logs.PrintAsJSON(list.itemsMap, t)
	logs.PrintAsJSON(list.allItemsMap, t)

	list.Delete(1)

	t.Log("===AFTER===")
	logs.PrintAsJSON(list.itemsMap, t)
	logs.PrintAsJSON(list.allItemsMap, t)
}

func TestGC(t *testing.T) {
	list := NewIPList()
	list.Add(&IPItem{
		Id:        1,
		IPFrom:    utils.IP2Long("192.168.1.100"),
		IPTo:      utils.IP2Long("192.168.1.101"),
		ExpiredAt: time.Now().Unix() + 1,
	})
	list.Add(&IPItem{
		Id:        2,
		IPFrom:    utils.IP2Long("192.168.1.102"),
		IPTo:      utils.IP2Long("192.168.1.103"),
		ExpiredAt: 0,
	})
	logs.PrintAsJSON(list.itemsMap, t)
	logs.PrintAsJSON(list.allItemsMap, t)

	time.Sleep(2 * time.Second)
	t.Log("===AFTER GC===")
	logs.PrintAsJSON(list.itemsMap, t)
	logs.PrintAsJSON(list.sortedItems, t)
}

func TestTooManyLists(t *testing.T) {
	debug.SetMaxThreads(20)

	var lists = []*IPList{}
	var locker = &sync.Mutex{}
	for i := 0; i < 1000; i++ {
		locker.Lock()
		lists = append(lists, NewIPList())
		locker.Unlock()
	}

	time.Sleep(1 * time.Second)
	t.Log(runtime.NumGoroutine())
	t.Log(len(lists), "lists")
}

func BenchmarkIPList_Contains(b *testing.B) {
	runtime.GOMAXPROCS(1)

	var list = NewIPList()
	for i := 1; i < 200_000; i++ {
		list.AddDelay(&IPItem{
			Id:        uint64(i),
			IPFrom:    utils.IP2Long(strconv.Itoa(rands.Int(0, 255)) + "." + strconv.Itoa(rands.Int(0, 255)) + ".0.1"),
			IPTo:      utils.IP2Long(strconv.Itoa(rands.Int(0, 255)) + "." + strconv.Itoa(rands.Int(0, 255)) + ".0.1"),
			ExpiredAt: time.Now().Unix() + 60,
		})
	}
	list.Sort()

	b.Log(len(list.itemsMap), "ip")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = list.Contains(utils.IP2Long("192.168.1.100"))
	}
}
