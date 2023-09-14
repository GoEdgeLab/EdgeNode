package caches

import (
	"fmt"
	"github.com/TeaOSLab/EdgeNode/internal/utils/testutils"
	"github.com/cespare/xxhash"
	"github.com/iwind/TeaGo/logs"
	"github.com/iwind/TeaGo/rands"
	"github.com/iwind/TeaGo/types"
	"math/rand"
	"sort"
	"strconv"
	"testing"
	"time"
)

func TestMemoryList_Add(t *testing.T) {
	list := NewMemoryList().(*MemoryList)
	_ = list.Init()
	_ = list.Add("a", &Item{
		Key:        "a1",
		ExpiredAt:  time.Now().Unix() + 3600,
		HeaderSize: 1024,
	})
	_ = list.Add("b", &Item{
		Key:        "b1",
		ExpiredAt:  time.Now().Unix() + 3600,
		HeaderSize: 1024,
	})
	_ = list.Add("123456", &Item{
		Key:        "c1",
		ExpiredAt:  time.Now().Unix() + 3600,
		HeaderSize: 1024,
	})
	t.Log(list.prefixes)
	logs.PrintAsJSON(list.itemMaps, t)
	t.Log(list.Count())
}

func TestMemoryList_Remove(t *testing.T) {
	list := NewMemoryList().(*MemoryList)
	_ = list.Init()
	_ = list.Add("a", &Item{
		Key:        "a1",
		ExpiredAt:  time.Now().Unix() + 3600,
		HeaderSize: 1024,
	})
	_ = list.Add("b", &Item{
		Key:        "b1",
		ExpiredAt:  time.Now().Unix() + 3600,
		HeaderSize: 1024,
	})
	_ = list.Remove("b")
	list.print(t)
	t.Log(list.Count())
}

func TestMemoryList_Purge(t *testing.T) {
	list := NewMemoryList().(*MemoryList)
	_ = list.Init()
	_ = list.Add("a", &Item{
		Key:        "a1",
		ExpiredAt:  time.Now().Unix() + 3600,
		HeaderSize: 1024,
	})
	_ = list.Add("b", &Item{
		Key:        "b1",
		ExpiredAt:  time.Now().Unix() + 3600,
		HeaderSize: 1024,
	})
	_ = list.Add("c", &Item{
		Key:        "c1",
		ExpiredAt:  time.Now().Unix() - 3600,
		HeaderSize: 1024,
	})
	_ = list.Add("d", &Item{
		Key:        "d1",
		ExpiredAt:  time.Now().Unix() - 2,
		HeaderSize: 1024,
	})
	_, _ = list.Purge(100, func(hash string) error {
		t.Log("delete:", hash)
		return nil
	})
	list.print(t)

	for i := 0; i < 1000; i++ {
		_, _ = list.Purge(100, func(hash string) error {
			t.Log("delete:", hash)
			return nil
		})
		t.Log(list.purgeIndex)
	}

	t.Log(list.Count())
}

func TestMemoryList_Purge_Large_List(t *testing.T) {
	list := NewMemoryList().(*MemoryList)
	_ = list.Init()

	for i := 0; i < 1_000_000; i++ {
		_ = list.Add("a"+strconv.Itoa(i), &Item{
			Key:        "a" + strconv.Itoa(i),
			ExpiredAt:  time.Now().Unix() + int64(rands.Int(0, 24*3600)),
			HeaderSize: 1024,
		})
	}

	if testutils.IsSingleTesting() {
		time.Sleep(1 * time.Hour)
	}
}

func TestMemoryList_Stat(t *testing.T) {
	list := NewMemoryList()
	_ = list.Init()
	_ = list.Add("a", &Item{
		Key:        "a1",
		ExpiredAt:  time.Now().Unix() + 3600,
		HeaderSize: 1024,
	})
	_ = list.Add("b", &Item{
		Key:        "b1",
		ExpiredAt:  time.Now().Unix() + 3600,
		HeaderSize: 1024,
	})
	_ = list.Add("c", &Item{
		Key:        "c1",
		ExpiredAt:  time.Now().Unix(),
		HeaderSize: 1024,
	})
	_ = list.Add("d", &Item{
		Key:        "d1",
		ExpiredAt:  time.Now().Unix() - 2,
		HeaderSize: 1024,
	})
	result, _ := list.Stat(func(hash string) bool {
		// 随机测试
		rand.Seed(time.Now().UnixNano())
		return rand.Int()%2 == 0
	})
	t.Log(result)
}

func TestMemoryList_CleanPrefix(t *testing.T) {
	list := NewMemoryList()
	_ = list.Init()
	before := time.Now()
	for i := 0; i < 1_000_000; i++ {
		key := "https://www.teaos.cn/hello/" + strconv.Itoa(i/10000) + "/" + strconv.Itoa(i) + ".html"
		_ = list.Add(fmt.Sprintf("%d", xxhash.Sum64String(key)), &Item{
			Key:        key,
			ExpiredAt:  time.Now().Unix() + 3600,
			BodySize:   0,
			HeaderSize: 0,
		})
	}
	t.Log(time.Since(before).Seconds()*1000, "ms")

	before = time.Now()
	err := list.CleanPrefix("https://www.teaos.cn/hello/10")
	if err != nil {
		t.Fatal(err)
	}

	logs.Println(list.Stat(func(hash string) bool {
		return true
	}))

	t.Log(time.Since(before).Seconds()*1000, "ms")
}

func TestMapRandomDelete(t *testing.T) {
	var countMap = map[int]int{} // k => count

	for j := 0; j < 1_000_000; j++ {
		var m = map[int]bool{}
		for i := 0; i < 100; i++ {
			m[i] = true
		}

		var count = 0
		for k := range m {
			delete(m, k)
			count++
			if count >= 10 {
				break
			}
		}

		for k := range m {
			countMap[k]++
		}
	}

	var counts = []int{}
	for _, count := range countMap {
		counts = append(counts, count)
	}
	sort.Ints(counts)
	t.Log("["+types.String(len(counts))+"]", counts)
}

func TestMemoryList_PurgeLFU(t *testing.T) {
	var list = NewMemoryList().(*MemoryList)

	var before = time.Now()
	defer func() {
		t.Log(time.Since(before).Seconds()*1000, "ms")
	}()

	_ = list.Add("1", &Item{})
	_ = list.Add("2", &Item{})
	_ = list.Add("3", &Item{})
	_ = list.Add("4", &Item{})
	_ = list.Add("5", &Item{})

	//_ = list.IncreaseHit("1")
	//_ = list.IncreaseHit("2")
	//_ = list.IncreaseHit("3")
	//_ = list.IncreaseHit("4")
	//_ = list.IncreaseHit("5")

	count, err := list.Count()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("count items before purge:", count)

	err = list.PurgeLFU(5, func(hash string) error {
		t.Log("purge lfu:", hash)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ok")

	count, err = list.Count()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("count items left:", count)
}

func TestMemoryList_CleanAll(t *testing.T) {
	var list = NewMemoryList().(*MemoryList)
	_ = list.Add("a", &Item{})
	_ = list.CleanAll()
	logs.PrintAsJSON(list.itemMaps, t)
	t.Log(list.Count())
}

func TestMemoryList_GC(t *testing.T) {
	list := NewMemoryList().(*MemoryList)
	_ = list.Init()
	for i := 0; i < 1_000_000; i++ {
		key := "https://www.teaos.cn/hello" + strconv.Itoa(i/100000) + "/" + strconv.Itoa(i) + ".html"
		_ = list.Add(fmt.Sprintf("%d", xxhash.Sum64String(key)), &Item{
			Key:        key,
			ExpiredAt:  0,
			BodySize:   0,
			HeaderSize: 0,
		})
	}
	time.Sleep(10 * time.Second)
	t.Log("clean...", len(list.itemMaps))
	_ = list.CleanAll()
	t.Log("cleanAll...", len(list.itemMaps))
	before := time.Now()
	//runtime.GC()
	t.Log("gc cost:", time.Since(before).Seconds()*1000, "ms")

	if testutils.IsSingleTesting() {
		timeout := time.NewTimer(2 * time.Minute)
		<-timeout.C
		t.Log("2 minutes passed")

		time.Sleep(30 * time.Minute)
	}
}
