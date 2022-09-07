// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package caches_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/caches"
	"github.com/iwind/TeaGo/Tea"
	_ "github.com/iwind/TeaGo/bootstrap"
	"testing"
	"time"
)

func TestFileListDB_ListLFUItems(t *testing.T) {
	var db = caches.NewFileListDB()
	err := db.Open(Tea.Root + "/data/cache-db-large.db")
	//err := db.Open(Tea.Root + "/data/cache-index/p1/db-0.db")
	if err != nil {
		t.Fatal(err)
	}
	err = db.Init()
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		_ = db.Close()
	}()

	hashList, err := db.ListLFUItems(100)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("[", len(hashList), "]", hashList)
}

func TestFileListDB_IncreaseHitAsync(t *testing.T) {
	var db = caches.NewFileListDB()
	err := db.Open(Tea.Root + "/data/cache-db-large.db")
	if err != nil {
		t.Fatal(err)
	}
	err = db.Init()
	err = db.IncreaseHitAsync("4598e5231ba47d6ec7aa9ea640ff2eaf")
	if err != nil {
		t.Fatal(err)
	}
	// wait transaction
	time.Sleep(1 * time.Second)
}
