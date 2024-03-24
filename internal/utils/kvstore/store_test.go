// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package kvstore_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils/kvstore"
	"github.com/cockroachdb/pebble"
	"github.com/iwind/TeaGo/Tea"
	_ "github.com/iwind/TeaGo/bootstrap"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	m.Run()

	if testingStore != nil {
		_ = testingStore.Close()
	}
}

func TestStore_Open(t *testing.T) {
	store, err := kvstore.OpenStore("test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = store.Close()
	}()

	t.Log("opened")
}

func TestStore_RawDB(t *testing.T) {
	store, err := kvstore.OpenStore("test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = store.Close()
	}()

	err = store.RawDB().Set([]byte("hello"), []byte("world"), nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestOpenStoreDir(t *testing.T) {
	store, err := kvstore.OpenStoreDir(Tea.Root+"/data/stores", "test3")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = store.Close()
	}()

	t.Log("opened")

	_ = store
}

func TestStore_CloseTwice(t *testing.T) {
	store, err := kvstore.OpenStore("test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		for i := 0; i < 3; i++ {
			err = store.Close()
			if err != nil {
				t.Fatal(err)
			}
		}
	}()
}

func TestStore_Count(t *testing.T) {
	testCountStore(t)
}

var testingStore *kvstore.Store

func testOpenStore(t *testing.T) *kvstore.DB {
	var err error
	testingStore, err = kvstore.OpenStore("test")
	if err != nil {
		t.Fatal(err)
	}

	db, err := testingStore.NewDB("db1")
	if err != nil {
		t.Fatal(err)
	}

	return db
}

func testOpenStoreTable[T any](t *testing.T, tableName string, encoder kvstore.ValueEncoder[T]) *kvstore.Table[T] {
	var err error

	var before = time.Now()
	testingStore, err = kvstore.OpenStore("test")
	if err != nil {
		t.Fatal(err)
	}
	t.Log("store open cost:", time.Since(before).Seconds()*1000, "ms")

	db, err := testingStore.NewDB("db1")
	if err != nil {
		t.Fatal(err)
	}

	table, err := kvstore.NewTable[T](tableName, encoder)
	if err != nil {
		t.Fatal(err)
	}
	db.AddTable(table)

	return table
}

func testOpenStoreTableForBenchmark[T any](t *testing.B, tableName string, encoder kvstore.ValueEncoder[T]) *kvstore.Table[T] {
	var err error
	testingStore, err = kvstore.OpenStore("test")
	if err != nil {
		t.Fatal(err)
	}

	db, err := testingStore.NewDB("db1")
	if err != nil {
		t.Fatal(err)
	}

	table, err := kvstore.NewTable[T](tableName, encoder)
	if err != nil {
		t.Fatal(err)
	}
	db.AddTable(table)

	return table
}

func testCountStore(t *testing.T) {
	var err error
	testingStore, err = kvstore.OpenStore("test")
	if err != nil {
		t.Fatal(err)
	}
	var count int
	it, err := testingStore.RawDB().NewIter(&pebble.IterOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = it.Close()
	}()
	for it.First(); it.Valid(); it.Next() {
		count++
	}
	t.Log("count:", count)
}
