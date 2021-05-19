// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package caches

import (
	"github.com/iwind/TeaGo/Tea"
	stringutil "github.com/iwind/TeaGo/utils/string"
	"strconv"
	"testing"
	"time"
)

func TestFileList_Init(t *testing.T) {
	list := NewFileList(Tea.Root + "/data")
	err := list.Init()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ok")
}

func TestFileList_Add(t *testing.T) {
	list := NewFileList(Tea.Root + "/data")
	err := list.Init()
	if err != nil {
		t.Fatal(err)
	}
	err = list.Add(stringutil.Md5("123456"), &Item{
		Key:        "123456",
		ExpiredAt:  time.Now().Unix(),
		HeaderSize: 1,
		MetaSize:   2,
		BodySize:   3,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ok")
}

func TestFileList_Add_Many(t *testing.T) {
	list := NewFileList(Tea.Root + "/data")
	err := list.Init()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 100_0000; i++ {
		u := "http://edge.teaos.cn/123456" + strconv.Itoa(i)
		err = list.Add(stringutil.Md5(u), &Item{
			Key:        u,
			ExpiredAt:  time.Now().Unix(),
			HeaderSize: 1,
			MetaSize:   2,
			BodySize:   3,
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	t.Log("ok")
}

func TestFileList_Exist(t *testing.T) {
	list := NewFileList(Tea.Root + "/data")
	err := list.Init()
	if err != nil {
		t.Fatal(err)
	}
	{
		exists, err := list.Exist(stringutil.Md5("123456"))
		if err != nil {
			t.Fatal(err)
		}
		t.Log("exists:", exists)
	}
	{
		exists, err := list.Exist(stringutil.Md5("654321"))
		if err != nil {
			t.Fatal(err)
		}
		t.Log("exists:", exists)
	}
}

func TestFileList_FindKeysWithPrefix(t *testing.T) {
	list := NewFileList(Tea.Root + "/data")
	err := list.Init()
	if err != nil {
		t.Fatal(err)
	}
	before := time.Now()
	keys, err := list.FindKeysWithPrefix("1234")
	if err != nil {
		t.Fatal(err)
	}
	t.Log("keys:", keys)
	t.Log(time.Since(before).Seconds()*1000, "ms")
}

func TestFileList_Remove(t *testing.T) {
	list := NewFileList(Tea.Root + "/data")
	err := list.Init()
	if err != nil {
		t.Fatal(err)
	}
	list.OnRemove(func(item *Item) {
		t.Logf("remove %#v", item)
	})
	err = list.Remove(stringutil.Md5("123456"))
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ok")
}

func TestFileList_Purge(t *testing.T) {
	list := NewFileList(Tea.Root + "/data")
	err := list.Init()
	if err != nil {
		t.Fatal(err)
	}
	err = list.Purge(2, func(hash string) error {
		t.Log(hash)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ok")
}

func TestFileList_Stat(t *testing.T) {
	list := NewFileList(Tea.Root + "/data")
	err := list.Init()
	if err != nil {
		t.Fatal(err)
	}
	stat, err := list.Stat(nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("count:", stat.Count, "size:", stat.Size, "valueSize:", stat.ValueSize)
}

func TestFileList_Count(t *testing.T) {
	list := NewFileList(Tea.Root + "/data")
	err := list.Init()
	if err != nil {
		t.Fatal(err)
	}
	before := time.Now()
	count, err := list.Count()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("count:", count)
	t.Log(time.Since(before).Seconds()*1000, "ms")
}

func TestFileList_CleanAll(t *testing.T) {
	list := NewFileList(Tea.Root + "/data")
	err := list.Init()
	if err != nil {
		t.Fatal(err)
	}
	err = list.CleanAll()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ok")
	t.Log(list.Count())
}
