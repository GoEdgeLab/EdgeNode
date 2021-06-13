// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package caches

import (
	"github.com/iwind/TeaGo/Tea"
	"github.com/iwind/TeaGo/rands"
	stringutil "github.com/iwind/TeaGo/utils/string"
	"strconv"
	"sync"
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
		Host:       "teaos.cn",
		ServerId:   1,
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
	before := time.Now()
	for i := 0; i < 2000_0000; i++ {
		u := "http://edge.teaos.cn/123456" + strconv.Itoa(i)
		_ = list.Add(stringutil.Md5(u), &Item{
			Key:        u,
			ExpiredAt:  time.Now().Unix() + 3600,
			HeaderSize: 1,
			MetaSize:   2,
			BodySize:   3,
		})
		if err != nil {
			t.Fatal(err)
		}
		if i > 0 && i%10_000 == 0 {
			t.Log(i, int(10000/time.Since(before).Seconds()), "qps")
			before = time.Now()
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
	before := time.Now()
	defer func() {
		t.Log(time.Since(before).Seconds()*1000, "ms")
	}()
	{
		exists, err := list.Exist(stringutil.Md5("123456"))
		if err != nil {
			t.Fatal(err)
		}
		t.Log("exists:", exists)
	}
	{
		exists, err := list.Exist(stringutil.Md5("http://edge.teaos.cn/1234561"))
		if err != nil {
			t.Fatal(err)
		}
		t.Log("exists:", exists)
	}
}

func TestFileList_Exist_Many_DB(t *testing.T) {
	// 测试在多个数据库下的性能
	var listSlice = []ListInterface{}
	for i := 1; i <= 10; i++ {
		list := NewFileList(Tea.Root + "/data/data" + strconv.Itoa(i))
		err := list.Init()
		if err != nil {
			t.Fatal(err)
		}
		listSlice = append(listSlice, list)
	}

	var wg = sync.WaitGroup{}
	var threads = 8
	wg.Add(threads)

	var count = 200_000
	var countLocker sync.Mutex
	var tasks = make(chan int, count)
	for i := 0; i < count; i++ {
		tasks <- i
	}

	var hash = stringutil.Md5("http://edge.teaos.cn/1234561")

	before := time.Now()
	defer func() {
		t.Log(time.Since(before).Seconds()*1000, "ms")
	}()

	for i := 0; i < threads; i++ {
		go func() {
			defer wg.Done()

			for {
				select {
				case <-tasks:
					countLocker.Lock()
					count--
					countLocker.Unlock()

					var list = listSlice[rands.Int(0, len(listSlice)-1)]
					_, _ = list.Exist(hash)
				default:
					return
				}
			}
		}()
	}
	wg.Wait()
	t.Log("left:", count)
}

func TestFileList_CleanPrefix(t *testing.T) {
	list := NewFileList(Tea.Root + "/data")
	err := list.Init()
	if err != nil {
		t.Fatal(err)
	}
	before := time.Now()
	err = list.CleanPrefix("1234")
	if err != nil {
		t.Fatal(err)
	}
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

func BenchmarkFileList_Exist(b *testing.B) {
	list := NewFileList(Tea.Root + "/data")
	err := list.Init()
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < b.N; i++ {
		_, _ = list.Exist("f0eb5b87e0b0041f3917002c0707475f")
	}
}
