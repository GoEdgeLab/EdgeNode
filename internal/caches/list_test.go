package caches

import (
	"math/rand"
	"testing"
	"time"
)

func TestList_Add(t *testing.T) {
	list := NewList()
	list.Add("a", &Item{
		Key:       "a1",
		ExpiredAt: time.Now().Unix() + 3600,
		Size:      1024,
	})
	list.Add("b", &Item{
		Key:       "b1",
		ExpiredAt: time.Now().Unix() + 3600,
		Size:      1024,
	})
	t.Log(list.m)
}

func TestList_Remove(t *testing.T) {
	list := NewList()
	list.Add("a", &Item{
		Key:       "a1",
		ExpiredAt: time.Now().Unix() + 3600,
		Size:      1024,
	})
	list.Add("b", &Item{
		Key:       "b1",
		ExpiredAt: time.Now().Unix() + 3600,
		Size:      1024,
	})
	list.Remove("b")
	t.Log(list.m)
}

func TestList_Purge(t *testing.T) {
	list := NewList()
	list.Add("a", &Item{
		Key:       "a1",
		ExpiredAt: time.Now().Unix() + 3600,
		Size:      1024,
	})
	list.Add("b", &Item{
		Key:       "b1",
		ExpiredAt: time.Now().Unix() + 3600,
		Size:      1024,
	})
	list.Add("c", &Item{
		Key:       "c1",
		ExpiredAt: time.Now().Unix() - 3600,
		Size:      1024,
	})
	list.Add("d", &Item{
		Key:       "d1",
		ExpiredAt: time.Now().Unix() - 2,
		Size:      1024,
	})
	list.Purge(100, func(hash string) {
		t.Log("delete:", hash)
	})
	t.Log(list.m)
}

func TestList_Stat(t *testing.T) {
	list := NewList()
	list.Add("a", &Item{
		Key:       "a1",
		ExpiredAt: time.Now().Unix() + 3600,
		Size:      1024,
	})
	list.Add("b", &Item{
		Key:       "b1",
		ExpiredAt: time.Now().Unix() + 3600,
		Size:      1024,
	})
	list.Add("c", &Item{
		Key:       "c1",
		ExpiredAt: time.Now().Unix(),
		Size:      1024,
	})
	list.Add("d", &Item{
		Key:       "d1",
		ExpiredAt: time.Now().Unix() - 2,
		Size:      1024,
	})
	result := list.Stat(func(hash string) bool {
		// 随机测试
		rand.Seed(time.Now().UnixNano())
		return rand.Int()%2 == 0
	})
	t.Log(result)
}
