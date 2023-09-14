// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package caches_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/caches"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/TeaOSLab/EdgeNode/internal/utils/testutils"
	"github.com/iwind/TeaGo/Tea"
	"github.com/iwind/TeaGo/rands"
	"github.com/iwind/TeaGo/types"
	stringutil "github.com/iwind/TeaGo/utils/string"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestFileList_Init(t *testing.T) {
	var list = caches.NewFileList(Tea.Root + "/data/cache-index/p1")

	defer func() {
		_ = list.Close()
	}()

	err := list.Init()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = list.Close()
	}()
	t.Log("ok")
}

func TestFileList_Add(t *testing.T) {
	var list = caches.NewFileList(Tea.Root + "/data/cache-index/p1").(*caches.FileList)

	defer func() {
		_ = list.Close()
	}()

	err := list.Init()
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		_ = list.Close()
	}()

	var hash = stringutil.Md5("123456")
	t.Log("db index:", list.GetDBIndex(hash))
	err = list.Add(hash, &caches.Item{
		Key:        "123456",
		ExpiredAt:  time.Now().Unix() + 1,
		HeaderSize: 1,
		MetaSize:   2,
		BodySize:   3,
		Host:       "teaos.cn",
		ServerId:   1,
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Log(list.Exist(hash))

	t.Log("ok")
}

func TestFileList_Add_Many(t *testing.T) {
	if !testutils.IsSingleTesting() {
		return
	}

	var list = caches.NewFileList(Tea.Root + "/data/cache-index/p1")

	defer func() {
		_ = list.Close()
	}()

	err := list.Init()
	if err != nil {
		t.Fatal(err)
	}

	var before = time.Now()
	for i := 0; i < 10_000_000; i++ {
		u := "https://edge.teaos.cn/123456" + strconv.Itoa(i)
		_ = list.Add(stringutil.Md5(u), &caches.Item{
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
	var list = caches.NewFileList(Tea.Root + "/data/cache-index/p1").(*caches.FileList)
	defer func() {
		_ = list.Close()
	}()

	err := list.Init()
	if err != nil {
		t.Fatal(err)
	}

	total, _ := list.Count()
	t.Log("total:", total)

	var before = time.Now()
	defer func() {
		t.Log(time.Since(before).Seconds()*1000, "ms")
	}()
	{
		var hash = stringutil.Md5("123456")
		exists, err := list.Exist(hash)
		if err != nil {
			t.Fatal(err)
		}
		t.Log(hash, "exists:", exists)
	}
	{
		var hash = stringutil.Md5("http://edge.teaos.cn/1234561")
		exists, err := list.Exist(hash)
		if err != nil {
			t.Fatal(err)
		}
		t.Log(hash, "exists:", exists)
	}
}

func TestFileList_Exist_Many_DB(t *testing.T) {
	// 测试在多个数据库下的性能
	var listSlice = []caches.ListInterface{}
	for i := 1; i <= 10; i++ {
		var list = caches.NewFileList(Tea.Root + "/data/data" + strconv.Itoa(i))
		err := list.Init()
		if err != nil {
			t.Fatal(err)
		}
		listSlice = append(listSlice, list)
	}

	defer func() {
		for _, list := range listSlice {
			_ = list.Close()
		}
	}()

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
		goman.New(func() {
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
		})
	}
	wg.Wait()
	t.Log("left:", count)
}

func TestFileList_CleanPrefix(t *testing.T) {
	var list = caches.NewFileList(Tea.Root + "/data/cache-index/p1")

	defer func() {
		_ = list.Close()
	}()

	err := list.Init()
	if err != nil {
		t.Fatal(err)
	}

	before := time.Now()
	err = list.CleanPrefix("123")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(time.Since(before).Seconds()*1000, "ms")
}

func TestFileList_Remove(t *testing.T) {
	var list = caches.NewFileList(Tea.Root + "/data/cache-index/p1").(*caches.FileList)
	defer func() {
		_ = list.Close()
	}()

	err := list.Init()
	if err != nil {
		t.Fatal(err)
	}

	list.OnRemove(func(item *caches.Item) {
		t.Logf("remove %#v", item)
	})
	err = list.Remove(stringutil.Md5("123456"))
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ok")

	t.Log("===count===")
	t.Log(list.Count())
}

func TestFileList_Purge(t *testing.T) {
	var list = caches.NewFileList(Tea.Root + "/data/cache-index/p1")

	defer func() {
		_ = list.Close()
	}()

	err := list.Init()
	if err != nil {
		t.Fatal(err)
	}

	var count = 0
	_, err = list.Purge(caches.CountFileDB*2, func(hash string) error {
		t.Log(hash)
		count++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ok, purged", count, "items")
}

func TestFileList_PurgeLFU(t *testing.T) {
	var list = caches.NewFileList(Tea.Root + "/data/cache-index/p1")

	defer func() {
		_ = list.Close()
	}()

	err := list.Init()
	if err != nil {
		t.Fatal(err)
	}

	var count = 0
	err = list.PurgeLFU(caches.CountFileDB*2, func(hash string) error {
		t.Log(hash)
		count++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ok, purged", count, "items")
}

func TestFileList_Stat(t *testing.T) {
	var list = caches.NewFileList(Tea.Root + "/data/cache-index/p1")

	defer func() {
		_ = list.Close()
	}()

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
	var list = caches.NewFileList(Tea.Root + "/data")

	defer func() {
		_ = list.Close()
	}()

	err := list.Init()
	if err != nil {
		t.Fatal(err)
	}
	var before = time.Now()
	count, err := list.Count()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("count:", count)
	t.Log(time.Since(before).Seconds()*1000, "ms")
}

func TestFileList_CleanAll(t *testing.T) {
	var list = caches.NewFileList(Tea.Root + "/data")

	defer func() {
		_ = list.Close()
	}()

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

func TestFileList_UpgradeV3(t *testing.T) {
	var list = caches.NewFileList(Tea.Root + "/data/cache-index/p43").(*caches.FileList)

	defer func() {
		_ = list.Close()
	}()

	err := list.Init()
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		_ = list.Close()
	}()

	err = list.UpgradeV3("/Users/WorkSpace/EdgeProject/EdgeCache/p43", false)
	if err != nil {
		t.Log(err)
		return
	}
	t.Log("ok")
}

func BenchmarkFileList_Exist(b *testing.B) {
	var list = caches.NewFileList(Tea.Root + "/data/cache-index/p1")

	defer func() {
		_ = list.Close()
	}()

	err := list.Init()
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = list.Exist("f0eb5b87e0b0041f3917002c0707475f" + types.String(i))
	}
}
