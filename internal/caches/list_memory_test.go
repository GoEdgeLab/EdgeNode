package caches

import (
	"fmt"
	"github.com/cespare/xxhash"
	"math/rand"
	"strconv"
	"testing"
	"time"
)

func TestMemoryList_Add(t *testing.T) {
	list := NewMemoryList().(*MemoryList)
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
	t.Log(list.m)
}

func TestMemoryList_Remove(t *testing.T) {
	list := NewMemoryList().(*MemoryList)
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
	t.Log(list.m)
}

func TestMemoryList_Purge(t *testing.T) {
	list := NewMemoryList().(*MemoryList)
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
	_ = list.Purge(100, func(hash string) error {
		t.Log("delete:", hash)
		return nil
	})
	t.Log(list.m)
}

func TestMemoryList_Stat(t *testing.T) {
	list := NewMemoryList()
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

func TestMemoryList_FindKeysWithPrefix(t *testing.T) {
	list := NewMemoryList()
	before := time.Now()
	for i := 0; i < 1_000_000; i++ {
		key := "http://www.teaos.cn/hello" + strconv.Itoa(i/100000) + "/" + strconv.Itoa(i) + ".html"
		_ = list.Add(fmt.Sprintf("%d", xxhash.Sum64String(key)), &Item{
			Key:        key,
			ExpiredAt:  0,
			BodySize:   0,
			HeaderSize: 0,
		})
	}
	t.Log(time.Since(before).Seconds()*1000, "ms")

	before = time.Now()
	keys, err := list.FindKeysWithPrefix("http://www.teaos.cn/hello/5000")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(len(keys))
	t.Log(time.Since(before).Seconds()*1000, "ms")
}

func TestMemoryList_GC(t *testing.T) {
	list := NewMemoryList().(*MemoryList)
	for i := 0; i < 1_000_000; i++ {
		key := "http://www.teaos.cn/hello" + strconv.Itoa(i/100000) + "/" + strconv.Itoa(i) + ".html"
		_ = list.Add(fmt.Sprintf("%d", xxhash.Sum64String(key)), &Item{
			Key:        key,
			ExpiredAt:  0,
			BodySize:   0,
			HeaderSize: 0,
		})
	}
	time.Sleep(10 * time.Second)
	t.Log("clean...", len(list.m))
	_ = list.CleanAll()
	before := time.Now()
	//runtime.GC()
	t.Log("gc cost:", time.Since(before).Seconds()*1000, "ms")

	timeout := time.NewTimer(2 * time.Minute)
	<-timeout.C
	t.Log("2 minutes passed")

	time.Sleep(30 * time.Minute)
}
