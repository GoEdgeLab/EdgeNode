// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package caches_test

import (
	"fmt"
	"github.com/TeaOSLab/EdgeNode/internal/caches"
	"github.com/TeaOSLab/EdgeNode/internal/utils/testutils"
	"github.com/iwind/TeaGo/Tea"
	_ "github.com/iwind/TeaGo/bootstrap"
	stringutil "github.com/iwind/TeaGo/utils/string"
	"math/rand"
	"strconv"
	"sync"
	"testing"
	"time"
)

var testingKVList *caches.KVFileList

func testOpenKVFileList(t *testing.T) *caches.KVFileList {
	var list = caches.NewKVFileList(Tea.Root + "/data/stores/cache-stores")
	err := list.Init()
	if err != nil {
		t.Fatal(err)
	}

	testingKVList = list
	return list
}

func TestNewKVFileList(t *testing.T) {
	var list = testOpenKVFileList(t)
	err := list.Close()
	if err != nil {
		t.Fatal(err)
	}
}

func TestKVFileList_Add(t *testing.T) {
	var list = testOpenKVFileList(t)
	defer func() {
		_ = list.Close()
	}()

	err := list.Add(stringutil.Md5("123456"), &caches.Item{
		Type:       caches.ItemTypeFile,
		Key:        "https://example.com/index.html",
		ExpiresAt:  time.Now().Unix() + 60,
		StaleAt:    0,
		HeaderSize: 0,
		BodySize:   4096,
		MetaSize:   0,
		Host:       "",
		ServerId:   1,
		Week:       0,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestKVFileList_Add_Many(t *testing.T) {
	if !testutils.IsSingleTesting() {
		return
	}

	var list = testOpenKVFileList(t)
	defer func() {
		_ = list.Close()
	}()

	const start = 0
	const count = 1_000_000
	const concurrent = 100

	var before = time.Now()
	defer func() {
		var costSeconds = time.Since(before).Seconds()
		t.Log("cost:", fmt.Sprintf("%.2fs", costSeconds), "qps:", fmt.Sprintf("%.2fK/s", float64(count)/1000/costSeconds))
	}()

	var wg = &sync.WaitGroup{}
	wg.Add(concurrent)
	for c := 0; c < concurrent; c++ {
		go func(c int) {
			defer wg.Done()

			var segmentStart = start + count/concurrent*c
			for i := segmentStart; i < segmentStart+count/concurrent; i++ {
				err := list.Add(stringutil.Md5(strconv.Itoa(i)), &caches.Item{
					Type:       caches.ItemTypeFile,
					Key:        "https://www.example.com/index.html" + strconv.Itoa(i),
					ExpiresAt:  time.Now().Unix() + 60,
					StaleAt:    0,
					HeaderSize: 0,
					BodySize:   int64(rand.Int() % 1_000_000),
					MetaSize:   0,
					Host:       "",
					ServerId:   1,
					Week:       0,
				})
				if err != nil {
					t.Log(err)
				}
			}
		}(c)
	}
	wg.Wait()
}

func TestKVFileList_Add_Many_Suffix(t *testing.T) {
	if !testutils.IsSingleTesting() {
		return
	}

	var list = testOpenKVFileList(t)
	defer func() {
		_ = list.Close()
	}()

	const start = 0
	const count = 1000
	const concurrent = 100

	var before = time.Now()
	defer func() {
		var costSeconds = time.Since(before).Seconds()
		t.Log("cost:", fmt.Sprintf("%.2fs", costSeconds), "qps:", fmt.Sprintf("%.2fK/s", float64(count)/1000/costSeconds))
	}()

	var wg = &sync.WaitGroup{}
	wg.Add(concurrent)
	for c := 0; c < concurrent; c++ {
		go func(c int) {
			defer wg.Done()

			var segmentStart = start + count/concurrent*c
			for i := segmentStart; i < segmentStart+count/concurrent; i++ {
				err := list.Add(stringutil.Md5(strconv.Itoa(i)+caches.SuffixAll), &caches.Item{
					Type:       caches.ItemTypeFile,
					Key:        "https://www.example.com/index.html" + strconv.Itoa(i) + caches.SuffixAll + "zip",
					ExpiresAt:  time.Now().Unix() + 60,
					StaleAt:    0,
					HeaderSize: 0,
					BodySize:   int64(rand.Int() % 1_000_000),
					MetaSize:   0,
					Host:       "",
					ServerId:   1,
					Week:       0,
				})
				if err != nil {
					t.Log(err)
				}
			}
		}(c)
	}
	wg.Wait()
}

func TestKVFileList_Exist(t *testing.T) {
	var list = testOpenKVFileList(t)
	defer func() {
		_ = list.Close()
	}()

	for _, hash := range []string{
		stringutil.Md5("123456"),
		stringutil.Md5("654321"),
	} {
		b, _, err := list.Exist(hash)
		if err != nil {
			t.Fatal(err)
		}
		t.Log(hash, "=>", b)
	}
}

func TestKVFileList_ExistQuick(t *testing.T) {
	var list = testOpenKVFileList(t)
	defer func() {
		_ = list.Close()
	}()

	for _, hash := range []string{
		stringutil.Md5("123456"),
		stringutil.Md5("654321"),
	} {
		b, err := list.ExistQuick(hash)
		if err != nil {
			t.Fatal(err)
		}
		t.Log(hash, "=>", b)
	}
}

func TestKVFileList_Remove(t *testing.T) {
	var list = testOpenKVFileList(t)
	defer func() {
		_ = list.Close()
	}()

	for _, hash := range []string{
		stringutil.Md5("123456"),
		stringutil.Md5("654321"),
	} {
		err := list.Remove(hash)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestKVFileList_CleanAll(t *testing.T) {
	var list = testOpenKVFileList(t)
	defer func() {
		_ = list.Close()
	}()

	err := list.CleanAll()
	if err != nil {
		t.Fatal(err)
	}
}

func TestKVFileList_Inspect(t *testing.T) {
	if !testutils.IsSingleTesting() {
		return
	}

	var list = testOpenKVFileList(t)
	defer func() {
		_ = list.Close()
	}()

	err := list.TestInspect(t)
	if err != nil {
		t.Fatal(err)
	}
}

func TestKVFileList_Purge(t *testing.T) {
	var list = testOpenKVFileList(t)
	defer func() {
		_ = list.Close()
	}()

	var before = time.Now()
	count, err := list.Purge(4_000, func(hash string) error {
		//t.Log("hash:", hash)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("cost:", fmt.Sprintf("%.2fms", time.Since(before).Seconds()*1000), "count:", count)
}

func TestKVFileList_PurgeLFU(t *testing.T) {
	var list = testOpenKVFileList(t)
	defer func() {
		_ = list.Close()
	}()

	var before = time.Now()
	err := list.PurgeLFU(20000, func(hash string) error {
		t.Log("hash:", hash)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("cost:", fmt.Sprintf("%.2fms", time.Since(before).Seconds()*1000))
}

func TestKVFileList_Count(t *testing.T) {
	var list = testOpenKVFileList(t)
	defer func() {
		_ = list.Close()
	}()

	var before = time.Now()
	count, err := list.Count()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("cost:", fmt.Sprintf("%.2fms", time.Since(before).Seconds()*1000), "count:", count)
}

func TestKVFileList_Stat(t *testing.T) {
	var list = testOpenKVFileList(t)
	defer func() {
		_ = list.Close()
	}()

	var before = time.Now()
	stat, err := list.Stat(func(hash string) bool {
		return true
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("cost:", fmt.Sprintf("%.2fms", time.Since(before).Seconds()*1000), "stat:", fmt.Sprintf("%+v", stat))
}

func TestKVFileList_CleanPrefix(t *testing.T) {
	var list = testOpenKVFileList(t)
	defer func() {
		_ = list.Close()
	}()

	var before = time.Now()

	defer func() {
		var costSeconds = time.Since(before).Seconds()
		t.Log("cost:", fmt.Sprintf("%.2fms", costSeconds*1000))
	}()

	err := list.CleanPrefix("https://www.example.com/index.html")
	if err != nil {
		t.Fatal(err)
	}
}

func TestKVFileList_CleanMatchPrefix(t *testing.T) {
	var list = testOpenKVFileList(t)
	defer func() {
		_ = list.Close()
	}()

	var before = time.Now()

	defer func() {
		var costSeconds = time.Since(before).Seconds()
		t.Log("cost:", fmt.Sprintf("%.2fms", costSeconds*1000))
	}()

	err := list.CleanMatchPrefix("https://*.example.com/index.html")
	if err != nil {
		t.Fatal(err)
	}
}

func TestKVFileList_CleanMatchKey(t *testing.T) {
	var list = testOpenKVFileList(t)
	defer func() {
		_ = list.Close()
	}()

	var before = time.Now()

	defer func() {
		var costSeconds = time.Since(before).Seconds()
		t.Log("cost:", fmt.Sprintf("%.2fms", costSeconds*1000))
	}()

	err := list.CleanMatchKey("https://*.example.com/index.html123")
	if err != nil {
		t.Fatal(err)
	}
}

func BenchmarkKVFileList_Exist(b *testing.B) {
	var list = caches.NewKVFileList(Tea.Root + "/data/stores/cache-stores")
	err := list.Init()
	if err != nil {
		b.Fatal(err)
	}

	defer func() {
		_ = list.Close()
	}()

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _, existErr := list.Exist(stringutil.Md5(strconv.Itoa(rand.Int() % 2_000_000)))
			if existErr != nil {
				b.Fatal(existErr)
			}
		}
	})
}
