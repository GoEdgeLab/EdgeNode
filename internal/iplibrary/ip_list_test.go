package iplibrary

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/iwind/TeaGo/assert"
	"github.com/iwind/TeaGo/logs"
	"runtime"
	"strconv"
	"testing"
	"time"
)

func TestIPList_Add_Empty(t *testing.T) {
	ipList := NewIPList()
	ipList.Add(&IPItem{
		Id: 1,
	})
	logs.PrintAsJSON(ipList.itemsMap, t)
	logs.PrintAsJSON(ipList.ipMap, t)
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
		IPFrom: utils.IP2Long("2001:db8:0:1::101"),
	})
	ipList.Add(&IPItem{
		Id:     4,
		IPFrom: 0,
		Type:   "all",
	})
	logs.PrintAsJSON(ipList.itemsMap, t)
	logs.PrintAsJSON(ipList.ipMap, t) // ip => items
}

func TestIPList_Update(t *testing.T) {
	ipList := NewIPList()
	ipList.Add(&IPItem{
		Id:     1,
		IPFrom: utils.IP2Long("192.168.1.1"),
	})
	/**ipList.Add(&IPItem{
		Id:     2,
		IPFrom: IP2Long("192.168.1.1"),
	})**/
	ipList.Add(&IPItem{
		Id:   1,
		IPTo: utils.IP2Long("192.168.1.2"),
	})
	logs.PrintAsJSON(ipList.itemsMap, t)
	logs.PrintAsJSON(ipList.ipMap, t)
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
	t.Log(len(ipList.ipMap), "ips")
	logs.PrintAsJSON(ipList.itemsMap, t)
	logs.PrintAsJSON(ipList.ipMap, t)
}

func TestIPList_Add_Overflow(t *testing.T) {
	a := assert.NewAssertion(t)

	ipList := NewIPList()
	ipList.Add(&IPItem{
		Id:     1,
		IPFrom: utils.IP2Long("192.168.1.1"),
		IPTo:   utils.IP2Long("192.169.255.1"),
	})
	t.Log(len(ipList.ipMap), "ips")
	a.IsTrue(len(ipList.ipMap) <= 65535)
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
	list := NewIPList()
	for i := 0; i < 255; i++ {
		list.Add(&IPItem{
			Id:        int64(i),
			IPFrom:    utils.IP2Long(strconv.Itoa(i) + ".168.0.1"),
			IPTo:      utils.IP2Long(strconv.Itoa(i) + ".168.255.1"),
			ExpiredAt: 0,
		})
	}
	t.Log(len(list.ipMap), "ip")

	before := time.Now()
	t.Log(list.Contains(utils.IP2Long("192.168.1.100")))
	t.Log(list.Contains(utils.IP2Long("192.168.2.100")))
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
	logs.PrintAsJSON(list.ipMap, t)

	list.Delete(1)

	t.Log("===AFTER===")
	logs.PrintAsJSON(list.itemsMap, t)
	logs.PrintAsJSON(list.ipMap, t)
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
	logs.PrintAsJSON(list.ipMap, t)

	time.Sleep(2 * time.Second)
	t.Log("===AFTER GC===")
	logs.PrintAsJSON(list.itemsMap, t)
	logs.PrintAsJSON(list.ipMap, t)
}

func BenchmarkIPList_Contains(b *testing.B) {
	runtime.GOMAXPROCS(1)

	list := NewIPList()
	for i := 192; i < 194; i++ {
		list.Add(&IPItem{
			Id:        int64(1),
			IPFrom:    utils.IP2Long(strconv.Itoa(i) + ".1.0.1"),
			IPTo:      utils.IP2Long(strconv.Itoa(i) + ".2.0.1"),
			ExpiredAt: time.Now().Unix() + 60,
		})
	}
	b.Log(len(list.ipMap), "ip")
	for i := 0; i < b.N; i++ {
		_ = list.Contains(utils.IP2Long("192.168.1.100"))
	}
}
