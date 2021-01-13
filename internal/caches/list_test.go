package caches

import (
	"fmt"
	"github.com/cespare/xxhash"
	"math/rand"
	"strconv"
	"testing"
	"time"
)

func TestList_Add(t *testing.T) {
	list := NewList()
	list.Add("a", &Item{
		Key:        "a1",
		ExpiredAt:  time.Now().Unix() + 3600,
		HeaderSize: 1024,
	})
	list.Add("b", &Item{
		Key:        "b1",
		ExpiredAt:  time.Now().Unix() + 3600,
		HeaderSize: 1024,
	})
	t.Log(list.m)
}

func TestList_Remove(t *testing.T) {
	list := NewList()
	list.Add("a", &Item{
		Key:        "a1",
		ExpiredAt:  time.Now().Unix() + 3600,
		HeaderSize: 1024,
	})
	list.Add("b", &Item{
		Key:        "b1",
		ExpiredAt:  time.Now().Unix() + 3600,
		HeaderSize: 1024,
	})
	list.Remove("b")
	t.Log(list.m)
}

func TestList_Purge(t *testing.T) {
	list := NewList()
	list.Add("a", &Item{
		Key:        "a1",
		ExpiredAt:  time.Now().Unix() + 3600,
		HeaderSize: 1024,
	})
	list.Add("b", &Item{
		Key:        "b1",
		ExpiredAt:  time.Now().Unix() + 3600,
		HeaderSize: 1024,
	})
	list.Add("c", &Item{
		Key:        "c1",
		ExpiredAt:  time.Now().Unix() - 3600,
		HeaderSize: 1024,
	})
	list.Add("d", &Item{
		Key:        "d1",
		ExpiredAt:  time.Now().Unix() - 2,
		HeaderSize: 1024,
	})
	list.Purge(100, func(hash string) {
		t.Log("delete:", hash)
	})
	t.Log(list.m)
}

func TestList_Stat(t *testing.T) {
	list := NewList()
	list.Add("a", &Item{
		Key:        "a1",
		ExpiredAt:  time.Now().Unix() + 3600,
		HeaderSize: 1024,
	})
	list.Add("b", &Item{
		Key:        "b1",
		ExpiredAt:  time.Now().Unix() + 3600,
		HeaderSize: 1024,
	})
	list.Add("c", &Item{
		Key:        "c1",
		ExpiredAt:  time.Now().Unix(),
		HeaderSize: 1024,
	})
	list.Add("d", &Item{
		Key:        "d1",
		ExpiredAt:  time.Now().Unix() - 2,
		HeaderSize: 1024,
	})
	result := list.Stat(func(hash string) bool {
		// 随机测试
		rand.Seed(time.Now().UnixNano())
		return rand.Int()%2 == 0
	})
	t.Log(result)
}

func TestList_FindKeysWithPrefix(t *testing.T) {
	list := NewList()
	before := time.Now()
	for i := 0; i < 1_000_000; i++ {
		key := "http://www.teaos.cn/hello" + strconv.Itoa(i/100000) + "/" + strconv.Itoa(i) + ".html"
		list.Add(fmt.Sprintf("%d", xxhash.Sum64String(key)), &Item{
			Key:        key,
			ExpiredAt:  0,
			BodySize:   0,
			HeaderSize: 0,
		})
	}
	t.Log(time.Since(before).Seconds()*1000, "ms")

	before = time.Now()
	keys := list.FindKeysWithPrefix("http://www.teaos.cn/hello/5000")
	t.Log(len(keys))
	t.Log(time.Since(before).Seconds()*1000, "ms")
}
