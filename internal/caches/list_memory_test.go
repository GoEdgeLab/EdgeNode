package caches_test

import (
	"fmt"
	"github.com/TeaOSLab/EdgeNode/internal/caches"
	"github.com/TeaOSLab/EdgeNode/internal/utils/testutils"
	"github.com/cespare/xxhash"
	"github.com/iwind/TeaGo/logs"
	"github.com/iwind/TeaGo/rands"
	"github.com/iwind/TeaGo/types"
	stringutil "github.com/iwind/TeaGo/utils/string"
	"math/rand"
	"sort"
	"strconv"
	"testing"
	"time"
)

func TestMemoryList_Add(t *testing.T) {
	list := caches.NewMemoryList().(*caches.MemoryList)
	_ = list.Init()
	_ = list.Add("a", &caches.Item{
		Key:        "a1",
		ExpiresAt:  time.Now().Unix() + 3600,
		HeaderSize: 1024,
	})
	_ = list.Add("b", &caches.Item{
		Key:        "b1",
		ExpiresAt:  time.Now().Unix() + 3600,
		HeaderSize: 1024,
	})
	_ = list.Add("123456", &caches.Item{
		Key:        "c1",
		ExpiresAt:  time.Now().Unix() + 3600,
		HeaderSize: 1024,
	})
	t.Log(list.Prefixes())
	logs.PrintAsJSON(list.ItemMaps(), t)
	t.Log(list.Count())
}

func TestMemoryList_Remove(t *testing.T) {
	list := caches.NewMemoryList().(*caches.MemoryList)
	_ = list.Init()
	_ = list.Add("a", &caches.Item{
		Key:        "a1",
		ExpiresAt:  time.Now().Unix() + 3600,
		HeaderSize: 1024,
	})
	_ = list.Add("b", &caches.Item{
		Key:        "b1",
		ExpiresAt:  time.Now().Unix() + 3600,
		HeaderSize: 1024,
	})
	_ = list.Remove("b")
	list.Print(t)
	t.Log(list.Count())
}

func TestMemoryList_Purge(t *testing.T) {
	list := caches.NewMemoryList().(*caches.MemoryList)
	_ = list.Init()
	_ = list.Add("a", &caches.Item{
		Key:        "a1",
		ExpiresAt:  time.Now().Unix() + 3600,
		HeaderSize: 1024,
	})
	_ = list.Add("b", &caches.Item{
		Key:        "b1",
		ExpiresAt:  time.Now().Unix() + 3600,
		HeaderSize: 1024,
	})
	_ = list.Add("c", &caches.Item{
		Key:        "c1",
		ExpiresAt:  time.Now().Unix() - 3600,
		HeaderSize: 1024,
	})
	_ = list.Add("d", &caches.Item{
		Key:        "d1",
		ExpiresAt:  time.Now().Unix() - 2,
		HeaderSize: 1024,
	})
	_, _ = list.Purge(100, func(hash string) error {
		t.Log("delete:", hash)
		return nil
	})
	list.Print(t)

	for i := 0; i < 1000; i++ {
		_, _ = list.Purge(100, func(hash string) error {
			t.Log("delete:", hash)
			return nil
		})
		t.Log(list.PurgeIndex())
	}

	t.Log(list.Count())
}

func TestMemoryList_Purge_Large_List(t *testing.T) {
	var list = caches.NewMemoryList().(*caches.MemoryList)
	_ = list.Init()

	var count = 100
	if testutils.IsSingleTesting() {
		count = 1_000_000
	}

	for i := 0; i < count; i++ {
		_ = list.Add("a"+strconv.Itoa(i), &caches.Item{
			Key:        "a" + strconv.Itoa(i),
			ExpiresAt:  time.Now().Unix() + int64(rands.Int(0, 24*3600)),
			HeaderSize: 1024,
		})
	}

	if testutils.IsSingleTesting() {
		time.Sleep(1 * time.Hour)
	}
}

func TestMemoryList_Stat(t *testing.T) {
	list := caches.NewMemoryList()
	_ = list.Init()
	_ = list.Add("a", &caches.Item{
		Key:        "a1",
		ExpiresAt:  time.Now().Unix() + 3600,
		HeaderSize: 1024,
	})
	_ = list.Add("b", &caches.Item{
		Key:        "b1",
		ExpiresAt:  time.Now().Unix() + 3600,
		HeaderSize: 1024,
	})
	_ = list.Add("c", &caches.Item{
		Key:        "c1",
		ExpiresAt:  time.Now().Unix(),
		HeaderSize: 1024,
	})
	_ = list.Add("d", &caches.Item{
		Key:        "d1",
		ExpiresAt:  time.Now().Unix() - 2,
		HeaderSize: 1024,
	})
	result, _ := list.Stat(func(hash string) bool {
		// 随机测试
		return rand.Int()%2 == 0
	})
	t.Log(result)
}

func TestMemoryList_CleanPrefix(t *testing.T) {
	list := caches.NewMemoryList()
	_ = list.Init()
	before := time.Now()
	var count = 100
	if testutils.IsSingleTesting() {
		count = 1_000_000
	}
	for i := 0; i < count; i++ {
		key := "https://www.teaos.cn/hello/" + strconv.Itoa(i/10000) + "/" + strconv.Itoa(i) + ".html"
		_ = list.Add(fmt.Sprintf("%d", xxhash.Sum64String(key)), &caches.Item{
			Key:        key,
			ExpiresAt:  time.Now().Unix() + 3600,
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

	var count = 1000
	if testutils.IsSingleTesting() {
		count = 1_000_000
	}

	for j := 0; j < count; j++ {
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
	var list = caches.NewMemoryList().(*caches.MemoryList)

	var before = time.Now()
	defer func() {
		t.Log(time.Since(before).Seconds()*1000, "ms")
	}()

	_ = list.Add("1", &caches.Item{})
	_ = list.Add("2", &caches.Item{})
	_ = list.Add("3", &caches.Item{})
	_ = list.Add("4", &caches.Item{})
	_ = list.Add("5", &caches.Item{})

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
	var list = caches.NewMemoryList().(*caches.MemoryList)
	_ = list.Add("a", &caches.Item{})
	_ = list.CleanAll()
	logs.PrintAsJSON(list.ItemMaps(), t)
	t.Log(list.Count())
}

func TestMemoryList_GC(t *testing.T) {
	if !testutils.IsSingleTesting() {
		return
	}

	list := caches.NewMemoryList().(*caches.MemoryList)
	_ = list.Init()
	for i := 0; i < 1_000_000; i++ {
		key := "https://www.teaos.cn/hello" + strconv.Itoa(i/100000) + "/" + strconv.Itoa(i) + ".html"
		_ = list.Add(fmt.Sprintf("%d", xxhash.Sum64String(key)), &caches.Item{
			Key:        key,
			ExpiresAt:  0,
			BodySize:   0,
			HeaderSize: 0,
		})
	}
	t.Log("clean...", len(list.ItemMaps()))
	_ = list.CleanAll()
	t.Log("cleanAll...", len(list.ItemMaps()))
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

func BenchmarkMemoryList(b *testing.B) {
	var list = caches.NewMemoryList()
	err := list.Init()
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < 1_000_000; i++ {
		_ = list.Add(stringutil.Md5(types.String(i)), &caches.Item{
			Key:        "a1",
			ExpiresAt:  time.Now().Unix() + 3600,
			HeaderSize: 1024,
		})
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = list.Exist(types.String("a" + types.String(rands.Int(1, 10000))))
			_ = list.Add("a"+types.String(rands.Int(1, 100000)), &caches.Item{})
			_, _ = list.Purge(1000, func(hash string) error {
				return nil
			})
		}
	})
}
