package iplibrary_test

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/iputils"
	"github.com/TeaOSLab/EdgeNode/internal/iplibrary"
	"github.com/TeaOSLab/EdgeNode/internal/utils/testutils"
	"github.com/iwind/TeaGo/assert"
	"math/rand"
	"runtime"
	"strconv"
	"testing"
	"time"
)

func TestIPItem_Contains(t *testing.T) {
	var a = assert.NewAssertion(t)

	{
		var item = &iplibrary.IPItem{
			IPFrom:    iputils.ToBytes("192.168.1.100"),
			IPTo:      nil,
			ExpiredAt: 0,
		}
		a.IsTrue(item.Contains(iputils.ToBytes("192.168.1.100")))
	}

	{
		var item = &iplibrary.IPItem{
			IPFrom:    iputils.ToBytes("192.168.1.100"),
			IPTo:      nil,
			ExpiredAt: time.Now().Unix() + 1,
		}
		a.IsTrue(item.Contains(iputils.ToBytes("192.168.1.100")))
	}

	{
		var item = &iplibrary.IPItem{
			IPFrom:    iputils.ToBytes("192.168.1.100"),
			IPTo:      nil,
			ExpiredAt: time.Now().Unix() - 1,
		}
		a.IsFalse(item.Contains(iputils.ToBytes("192.168.1.100")))
	}
	{
		var item = &iplibrary.IPItem{
			IPFrom:    iputils.ToBytes("192.168.1.100"),
			IPTo:      nil,
			ExpiredAt: 0,
		}
		a.IsFalse(item.Contains(iputils.ToBytes("192.168.1.101")))
	}

	{
		var item = &iplibrary.IPItem{
			IPFrom:    iputils.ToBytes("192.168.1.1"),
			IPTo:      iputils.ToBytes("192.168.1.101"),
			ExpiredAt: 0,
		}
		a.IsTrue(item.Contains(iputils.ToBytes("192.168.1.100")))
	}

	{
		var item = &iplibrary.IPItem{
			IPFrom:    iputils.ToBytes("192.168.1.1"),
			IPTo:      iputils.ToBytes("192.168.1.100"),
			ExpiredAt: 0,
		}
		a.IsTrue(item.Contains(iputils.ToBytes("192.168.1.100")))
	}

	{
		var item = &iplibrary.IPItem{
			IPFrom:    iputils.ToBytes("192.168.1.1"),
			IPTo:      iputils.ToBytes("192.168.1.101"),
			ExpiredAt: 0,
		}
		a.IsTrue(item.Contains(iputils.ToBytes("192.168.1.1")))
	}
}

func TestIPItem_Memory(t *testing.T) {
	var isSingleTest = testutils.IsSingleTesting()

	var list = iplibrary.NewIPList()
	var count = 100
	if isSingleTest {
		count = 2_000_000
	}
	for i := 0; i < count; i++ {
		list.Add(&iplibrary.IPItem{
			Type:       "ip",
			Id:         uint64(i),
			IPFrom:     iputils.ToBytes("192.168.1.1"),
			IPTo:       nil,
			ExpiredAt:  time.Now().Unix(),
			EventLevel: "",
		})
	}

	runtime.GC()

	t.Log("waiting")
	if isSingleTest {
		time.Sleep(10 * time.Second)
	}
}

func BenchmarkIPItem_Contains(b *testing.B) {
	runtime.GOMAXPROCS(1)

	var item = &iplibrary.IPItem{
		IPFrom:    iputils.ToBytes("192.168.1.1"),
		IPTo:      iputils.ToBytes("192.168.1.101"),
		ExpiredAt: 0,
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var ip = iputils.ToBytes("192.168.1." + strconv.Itoa(rand.Int()%255))
			item.Contains(ip)
		}
	})
}
