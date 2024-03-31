// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package kvstore_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils/kvstore"
	"github.com/cockroachdb/pebble"
	"github.com/iwind/TeaGo/Tea"
	"github.com/iwind/TeaGo/assert"
	_ "github.com/iwind/TeaGo/bootstrap"
	"sync"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	m.Run()

	if testingStore != nil {
		_ = testingStore.Close()
	}
}

func TestStore_Default(t *testing.T) {
	var a = assert.NewAssertion(t)

	store, err := kvstore.DefaultStore()
	if err != nil {
		t.Fatal(err)
	}
	a.IsTrue(store != nil)
}

func TestStore_Default_Concurrent(t *testing.T) {
	var lastStore *kvstore.Store

	const threads = 32

	var wg = &sync.WaitGroup{}
	wg.Add(threads)
	for i := 0; i < threads; i++ {
		go func() {
			defer wg.Done()

			store, err := kvstore.DefaultStore()
			if err != nil {
				t.Log("ERROR", err)
				t.Fail()
			}

			if lastStore != nil && lastStore != store {
				t.Log("ERROR", "should be single instance")
				t.Fail()
			}

			lastStore = store
		}()
	}
	wg.Wait()
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
	_ = store
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
