package iplibrary_test

import (
	"fmt"
	"github.com/TeaOSLab/EdgeCommon/pkg/iputils"
	"github.com/TeaOSLab/EdgeNode/internal/iplibrary"
	"github.com/TeaOSLab/EdgeNode/internal/utils/fasttime"
	"github.com/TeaOSLab/EdgeNode/internal/utils/testutils"
	"github.com/iwind/TeaGo/assert"
	"github.com/iwind/TeaGo/logs"
	"github.com/iwind/TeaGo/rands"
	"math/rand"
	"runtime"
	"runtime/debug"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestIPList_Add_Empty(t *testing.T) {
	var ipList = iplibrary.NewIPList()
	ipList.Add(&iplibrary.IPItem{
		Id: 1,
	})
	logs.PrintAsJSON(ipList.ItemsMap(), t)
	logs.PrintAsJSON(ipList.AllItemsMap(), t)
	logs.PrintAsJSON(ipList.IPMap(), t)
}

func TestIPList_Add_One(t *testing.T) {
	var a = assert.NewAssertion(t)

	var ipList = iplibrary.NewIPList()
	ipList.Add(&iplibrary.IPItem{
		Id:     1,
		IPFrom: iputils.ToBytes("192.168.1.1"),
	})
	ipList.Add(&iplibrary.IPItem{
		Id:   2,
		IPTo: iputils.ToBytes("192.168.1.2"),
	})
	ipList.Add(&iplibrary.IPItem{
		Id:     3,
		IPFrom: iputils.ToBytes("192.168.0.2"),
	})
	ipList.Add(&iplibrary.IPItem{
		Id:     4,
		IPFrom: iputils.ToBytes("192.168.0.2"),
		IPTo:   iputils.ToBytes("192.168.0.1"),
	})
	ipList.Add(&iplibrary.IPItem{
		Id:     5,
		IPFrom: iputils.ToBytes("2001:db8:0:1::101"),
	})
	ipList.Add(&iplibrary.IPItem{
		Id:     6,
		IPFrom: nil,
		Type:   "all",
	})
	t.Log("===items===")
	logs.PrintAsJSON(ipList.ItemsMap(), t)

	t.Log("===sorted items===")
	logs.PrintAsJSON(ipList.SortedRangeItems(), t)

	t.Log("===all items===")
	a.IsTrue(len(ipList.AllItemsMap()) == 1)
	logs.PrintAsJSON(ipList.AllItemsMap(), t) // ip => items

	t.Log("===ip items===")
	logs.PrintAsJSON(ipList.IPMap())
}

func TestIPList_Update(t *testing.T) {
	var ipList = iplibrary.NewIPList()
	ipList.Add(&iplibrary.IPItem{
		Id:     1,
		IPFrom: iputils.ToBytes("192.168.1.1"),
	})

	t.Log("===before===")
	logs.PrintAsJSON(ipList.ItemsMap(), t)
	logs.PrintAsJSON(ipList.SortedRangeItems(), t)
	logs.PrintAsJSON(ipList.IPMap(), t)

	/**ipList.Add(&iplibrary.IPItem{
		Id:     2,
		IPFrom: iputils.ToBytes("192.168.1.1"),
	})**/
	ipList.Add(&iplibrary.IPItem{
		Id: 1,
		//IPFrom: 123,
		IPTo: iputils.ToBytes("192.168.1.2"),
	})

	t.Log("===after===")
	logs.PrintAsJSON(ipList.ItemsMap(), t)
	logs.PrintAsJSON(ipList.SortedRangeItems(), t)
	logs.PrintAsJSON(ipList.IPMap(), t)
}

func TestIPList_Update_AllItems(t *testing.T) {
	var ipList = iplibrary.NewIPList()
	ipList.Add(&iplibrary.IPItem{
		Id:     1,
		Type:   iplibrary.IPItemTypeAll,
		IPFrom: nil,
	})
	ipList.Add(&iplibrary.IPItem{
		Id:   1,
		IPTo: nil,
	})
	t.Log("===items map===")
	logs.PrintAsJSON(ipList.ItemsMap(), t)
	t.Log("===all items map===")
	logs.PrintAsJSON(ipList.AllItemsMap(), t)
	t.Log("===ip map===")
	logs.PrintAsJSON(ipList.IPMap())
}

func TestIPList_Add_Range(t *testing.T) {
	var a = assert.NewAssertion(t)

	var ipList = iplibrary.NewIPList()
	ipList.Add(&iplibrary.IPItem{
		Id:     1,
		IPFrom: iputils.ToBytes("192.168.1.1"),
		IPTo:   iputils.ToBytes("192.168.2.1"),
	})
	ipList.Add(&iplibrary.IPItem{
		Id:   2,
		IPTo: iputils.ToBytes("192.168.1.2"),
	})
	ipList.Add(&iplibrary.IPItem{
		Id:     3,
		IPFrom: iputils.ToBytes("192.168.0.1"),
		IPTo:   iputils.ToBytes("192.168.0.2"),
	})

	a.IsTrue(len(ipList.SortedRangeItems()) == 2)

	t.Log(len(ipList.ItemsMap()), "ips")
	t.Log("===items map===")
	logs.PrintAsJSON(ipList.ItemsMap(), t)
	t.Log("===sorted range items===")
	logs.PrintAsJSON(ipList.SortedRangeItems())
	t.Log("===all items map===")
	logs.PrintAsJSON(ipList.AllItemsMap(), t)

	t.Log("===ip map===")
	logs.PrintAsJSON(ipList.IPMap(), t)
}

func TestNewIPList_Memory(t *testing.T) {
	var list = iplibrary.NewIPList()

	var count = 100
	if testutils.IsSingleTesting() {
		count = 2_000_000
	}
	var stat1 = testutils.ReadMemoryStat()

	for i := 0; i < count; i++ {
		list.AddDelay(&iplibrary.IPItem{
			Id:        uint64(i),
			IPFrom:    iputils.ToBytes(testutils.RandIP()),
			IPTo:      iputils.ToBytes(testutils.RandIP()),
			ExpiredAt: time.Now().Unix(),
		})
	}

	list.Sort()

	runtime.GC()

	var stat2 = testutils.ReadMemoryStat()
	t.Log((stat2.HeapInuse-stat1.HeapInuse)>>20, "MB")
}

func TestIPList_Contains(t *testing.T) {
	var a = assert.NewAssertion(t)

	var list = iplibrary.NewIPList()
	for i := 0; i < 255; i++ {
		list.Add(&iplibrary.IPItem{
			Id:        uint64(i),
			IPFrom:    iputils.ToBytes(strconv.Itoa(i) + ".168.0.1"),
			IPTo:      iputils.ToBytes(strconv.Itoa(i) + ".168.255.1"),
			ExpiredAt: 0,
		})
	}
	for i := 0; i < 255; i++ {
		list.Add(&iplibrary.IPItem{
			Id:     uint64(1000 + i),
			IPFrom: iputils.ToBytes("192.167.2." + strconv.Itoa(i)),
		})
	}

	list.Add(&iplibrary.IPItem{
		Id:     10000,
		IPFrom: iputils.ToBytes("::1"),
	})
	list.Add(&iplibrary.IPItem{
		Id:     10001,
		IPFrom: iputils.ToBytes("::2"),
		IPTo:   iputils.ToBytes("::5"),
	})

	t.Log(len(list.ItemsMap()), "ip")

	var before = time.Now()
	a.IsTrue(list.Contains(iputils.ToBytes("192.168.1.100")))
	a.IsTrue(list.Contains(iputils.ToBytes("192.168.2.100")))
	a.IsFalse(list.Contains(iputils.ToBytes("192.169.3.100")))
	a.IsFalse(list.Contains(iputils.ToBytes("192.167.3.100")))
	a.IsTrue(list.Contains(iputils.ToBytes("192.167.2.100")))
	a.IsTrue(list.Contains(iputils.ToBytes("::1")))
	a.IsTrue(list.Contains(iputils.ToBytes("::3")))
	a.IsFalse(list.Contains(iputils.ToBytes("::8")))
	t.Log(time.Since(before).Seconds()*1000, "ms")
}

func TestIPList_Contains_Many(t *testing.T) {
	var list = iplibrary.NewIPList()
	for i := 0; i < 1_000_000; i++ {
		list.AddDelay(&iplibrary.IPItem{
			Id:        uint64(i),
			IPFrom:    iputils.ToBytes(strconv.Itoa(rands.Int(0, 255)) + "." + strconv.Itoa(rands.Int(0, 255)) + "." + strconv.Itoa(rands.Int(0, 255)) + "." + strconv.Itoa(rands.Int(0, 255))),
			IPTo:      iputils.ToBytes(strconv.Itoa(rands.Int(0, 255)) + "." + strconv.Itoa(rands.Int(0, 255)) + "." + strconv.Itoa(rands.Int(0, 255)) + "." + strconv.Itoa(rands.Int(0, 255))),
			ExpiredAt: 0,
		})
	}

	var before = time.Now()
	list.Sort()
	t.Log("sort cost:", time.Since(before).Seconds()*1000, "ms")
	t.Log(len(list.ItemsMap()), "ip")

	before = time.Now()
	_ = list.Contains(iputils.ToBytes("192.168.1.100"))
	t.Log("contains cost:", time.Since(before).Seconds()*1000, "ms")
}

func TestIPList_ContainsAll(t *testing.T) {
	var a = assert.NewAssertion(t)

	{
		var list = iplibrary.NewIPList()
		list.Add(&iplibrary.IPItem{
			Id:     1,
			Type:   "all",
			IPFrom: nil,
		})
		var b = list.Contains(iputils.ToBytes("192.168.1.1"))
		a.IsTrue(b)

		list.Delete(1)

		b = list.Contains(iputils.ToBytes("192.168.1.1"))
		a.IsFalse(b)
	}

	{
		var list = iplibrary.NewIPList()
		list.Add(&iplibrary.IPItem{
			Id:     1,
			Type:   "all",
			IPFrom: iputils.ToBytes("0.0.0.0"),
		})
		var b = list.Contains(iputils.ToBytes("192.168.1.1"))
		a.IsTrue(b)

		list.Delete(1)

		b = list.Contains(iputils.ToBytes("192.168.1.1"))
		a.IsFalse(b)
	}
}

func TestIPList_ContainsIPStrings(t *testing.T) {
	var a = assert.NewAssertion(t)

	var list = iplibrary.NewIPList()
	for i := 0; i < 255; i++ {
		list.Add(&iplibrary.IPItem{
			Id:        uint64(i),
			IPFrom:    iputils.ToBytes(strconv.Itoa(i) + ".168.0.1"),
			IPTo:      iputils.ToBytes(strconv.Itoa(i) + ".168.255.1"),
			ExpiredAt: 0,
		})
	}
	t.Log(len(list.ItemsMap()), "ip")

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
	var list = iplibrary.NewIPList()
	list.Add(&iplibrary.IPItem{
		Id:        1,
		IPFrom:    iputils.ToBytes("192.168.0.1"),
		ExpiredAt: 0,
	})
	list.Add(&iplibrary.IPItem{
		Id:        2,
		IPFrom:    iputils.ToBytes("192.168.0.1"),
		ExpiredAt: 0,
	})
	list.Add(&iplibrary.IPItem{
		Id:        3,
		IPFrom:    iputils.ToBytes("192.168.1.1"),
		IPTo:      iputils.ToBytes("192.168.2.1"),
		ExpiredAt: 0,
	})
	t.Log("===before===")
	logs.PrintAsJSON(list.ItemsMap(), t)
	logs.PrintAsJSON(list.AllItemsMap(), t)
	logs.PrintAsJSON(list.SortedRangeItems())
	logs.PrintAsJSON(list.IPMap(), t)

	{
		var found bool
		for _, item := range list.SortedRangeItems() {
			if item.Id == 3 {
				found = true
				break
			}
		}
		if !found {
			t.Fatal("should be found")
		}
	}

	list.Delete(1)

	t.Log("===after===")
	logs.PrintAsJSON(list.ItemsMap(), t)
	logs.PrintAsJSON(list.AllItemsMap(), t)
	logs.PrintAsJSON(list.SortedRangeItems())
	logs.PrintAsJSON(list.IPMap(), t)

	list.Delete(3)

	{
		var found bool
		for _, item := range list.SortedRangeItems() {
			if item.Id == 3 {
				found = true
				break
			}
		}
		if found {
			t.Fatal("should be not found")
		}
	}
}

func TestIPList_GC(t *testing.T) {
	var a = assert.NewAssertion(t)

	var list = iplibrary.NewIPList()
	list.Add(&iplibrary.IPItem{
		Id:        1,
		IPFrom:    iputils.ToBytes("192.168.1.100"),
		IPTo:      iputils.ToBytes("192.168.1.101"),
		ExpiredAt: time.Now().Unix() + 1,
	})
	list.Add(&iplibrary.IPItem{
		Id:        2,
		IPFrom:    iputils.ToBytes("192.168.1.102"),
		IPTo:      iputils.ToBytes("192.168.1.103"),
		ExpiredAt: 0,
	})
	logs.PrintAsJSON(list.ItemsMap(), t)
	logs.PrintAsJSON(list.AllItemsMap(), t)

	time.Sleep(3 * time.Second)

	t.Log("===AFTER GC===")
	logs.PrintAsJSON(list.ItemsMap(), t)
	logs.PrintAsJSON(list.SortedRangeItems(), t)

	a.IsTrue(len(list.ItemsMap()) == 1)
	a.IsTrue(len(list.SortedRangeItems()) == 1)
}

func TestManyLists(t *testing.T) {
	debug.SetMaxThreads(20)

	var lists = []*iplibrary.IPList{}
	var locker = &sync.Mutex{}
	for i := 0; i < 1000; i++ {
		locker.Lock()
		lists = append(lists, iplibrary.NewIPList())
		locker.Unlock()
	}

	if testutils.IsSingleTesting() {
		time.Sleep(3 * time.Second)
	}
	t.Log(runtime.NumGoroutine())
	t.Log(len(lists), "lists")
}

func BenchmarkIPList_Add(b *testing.B) {
	runtime.GOMAXPROCS(1)

	var list = iplibrary.NewIPList()
	for i := 1; i < 200_000; i++ {
		list.AddDelay(&iplibrary.IPItem{
			Id:        uint64(i),
			IPFrom:    iputils.ToBytes(strconv.Itoa(rands.Int(0, 255)) + "." + strconv.Itoa(rands.Int(0, 255)) + ".0.1"),
			IPTo:      iputils.ToBytes(strconv.Itoa(rands.Int(0, 255)) + "." + strconv.Itoa(rands.Int(0, 255)) + ".0.1"),
			ExpiredAt: time.Now().Unix() + 60,
		})
	}

	list.Sort()

	b.Log(len(list.ItemsMap()), "ip")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var ip = fmt.Sprintf("%d.%d.%d.%d", rand.Int()%255, rand.Int()%255, rand.Int()%255, rand.Int()%255)
		list.Add(&iplibrary.IPItem{
			Type:       "",
			Id:         uint64(i % 1_000_000),
			IPFrom:     iputils.ToBytes(ip),
			IPTo:       nil,
			ExpiredAt:  fasttime.Now().Unix() + 3600,
			EventLevel: "",
		})
	}
}

func BenchmarkIPList_Contains(b *testing.B) {
	runtime.GOMAXPROCS(1)

	var list = iplibrary.NewIPList()
	for i := 1; i < 1_000_000; i++ {
		var item = &iplibrary.IPItem{
			Id:        uint64(i),
			IPFrom:    iputils.ToBytes(strconv.Itoa(rands.Int(0, 255)) + "." + strconv.Itoa(rands.Int(0, 255)) + ".0.1"),
			ExpiredAt: time.Now().Unix() + 60,
		}
		if i%100 == 0 {
			item.IPTo = iputils.ToBytes(strconv.Itoa(rands.Int(0, 255)) + "." + strconv.Itoa(rands.Int(0, 255)) + ".0.1")
		}
		list.Add(item)
	}

	//b.Log(len(list.ItemsMap()), "ip")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = list.Contains(iputils.ToBytes(testutils.RandIP()))
		}
	})
}

func BenchmarkIPList_Sort(b *testing.B) {
	var list = iplibrary.NewIPList()
	for i := 0; i < 1_000_000; i++ {
		var item = &iplibrary.IPItem{
			Id:        uint64(i),
			IPFrom:    iputils.ToBytes(strconv.Itoa(rands.Int(0, 255)) + "." + strconv.Itoa(rands.Int(0, 255)) + ".0.1"),
			ExpiredAt: time.Now().Unix() + 60,
		}

		if i%100 == 0 {
			item.IPTo = iputils.ToBytes(strconv.Itoa(rands.Int(0, 255)) + "." + strconv.Itoa(rands.Int(0, 255)) + ".0.1")
		}

		list.AddDelay(item)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			list.Sort()
		}
	})
}
