package iplibrary

import (
	"runtime"
	"strconv"
	"testing"
	"time"
)

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
			IPFrom:    IP2Long("192.168.1." + strconv.Itoa(i)),
			IPTo:      0,
			ExpiredAt: 0,
		})
	}
	t.Log(list.Contains(IP2Long("192.168.1.100")))
	t.Log(list.Contains(IP2Long("192.168.2.100")))
}

func BenchmarkIPList_Contains(b *testing.B) {
	runtime.GOMAXPROCS(1)

	list := NewIPList()
	for i := 0; i < 10_000; i++ {
		list.Add(&IPItem{
			Id:        int64(i),
			IPFrom:    IP2Long("192.168.1." + strconv.Itoa(i)),
			IPTo:      0,
			ExpiredAt: time.Now().Unix() + 60,
		})
	}
	for i := 0; i < b.N; i++ {
		_ = list.Contains(IP2Long("192.168.1.100"))
	}
}
