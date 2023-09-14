// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package caches_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/caches"
	"github.com/iwind/TeaGo/Tea"
	_ "github.com/iwind/TeaGo/bootstrap"
	"testing"
)

func TestFileListDB_ListLFUItems(t *testing.T) {
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
