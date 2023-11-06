// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package caches_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/caches"
	"github.com/TeaOSLab/EdgeNode/internal/utils/testutils"
	"github.com/iwind/TeaGo/Tea"
	_ "github.com/iwind/TeaGo/bootstrap"
	"runtime"
	"runtime/debug"
	"testing"
	"time"
)

func TestFileListDB_ListLFUItems(t *testing.T) {
	if !testutils.IsSingleTesting() {
		return
	}

	var db = caches.NewFileListDB()

	defer func() {
		_ = db.Close()
	}()

	err := db.Open(Tea.Root + "/data/cache-db-large.db")
	//err := db.Open(Tea.Root + "/data/cache-index/p1/db-0.db")
	if err != nil {
		t.Fatal(err)
	}
	err = db.Init()
	if err != nil {
		t.Fatal(err)
	}

	hashList, err := db.ListLFUItems(100)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("[", len(hashList), "]", hashList)
}

func TestFileListDB_CleanMatchKey(t *testing.T) {
	if !testutils.IsSingleTesting() {
		return
	}

	var db = caches.NewFileListDB()

	defer func() {
		_ = db.Close()
	}()

	err := db.Open(Tea.Root + "/data/cache-db-large.db")
	if err != nil {
		t.Fatal(err)
	}

	err = db.Init()
	if err != nil {
		t.Fatal(err)
	}

	err = db.CleanMatchKey("https://*.goedge.cn/large-text")
	if err != nil {
		t.Fatal(err)
	}

	err = db.CleanMatchKey("https://*.goedge.cn:1234/large-text?%2B____")
	if err != nil {
		t.Fatal(err)
	}
}

func TestFileListDB_CleanMatchPrefix(t *testing.T) {
	if !testutils.IsSingleTesting() {
		return
	}

	var db = caches.NewFileListDB()

	defer func() {
		_ = db.Close()
	}()

	err := db.Open(Tea.Root + "/data/cache-db-large.db")
	if err != nil {
		t.Fatal(err)
	}

	err = db.Init()
	if err != nil {
		t.Fatal(err)
	}

	err = db.CleanMatchPrefix("https://*.goedge.cn/large-text")
	if err != nil {
		t.Fatal(err)
	}

	err = db.CleanMatchPrefix("https://*.goedge.cn:1234/large-text?%2B____")
	if err != nil {
		t.Fatal(err)
	}
}

func TestFileListDB_Memory(t *testing.T) {
	if !testutils.IsSingleTesting() {
		return
	}

	var db = caches.NewFileListDB()

	defer func() {
		_ = db.Close()
	}()

	err := db.Open(Tea.Root + "/data/cache-index/p1/db-0.db")
	if err != nil {
		t.Fatal(err)
	}

	err = db.Init()
	if err != nil {
		t.Fatal(err)
	}

	t.Log(db.Total())

	// load hashes
	var maxId int64
	var hashList []string
	var before = time.Now()
	for i := 0; i < 1_000; i++ {
		hashList, maxId, err = db.ListHashes(maxId)
		if err != nil {
			t.Fatal(err)
		}
		if len(hashList) == 0 {
			t.Log("hashes loaded", time.Since(before).Seconds()*1000, "ms")
			break
		}
		if i%100 == 0 {
			t.Log(i)
		}
	}

	runtime.GC()
	debug.FreeOSMemory()

	//time.Sleep(600 * time.Second)

	for i := 0; i < 1_000; i++ {
		_, err = db.ListLFUItems(5000)
		if err != nil {
			t.Fatal(err)
		}
		if i%100 == 0 {
			t.Log(i)
		}
	}

	t.Log("loaded")

	runtime.GC()
	debug.FreeOSMemory()

	time.Sleep(600 * time.Second)
}
