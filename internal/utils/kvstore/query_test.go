// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package kvstore_test

import (
	"fmt"
	"github.com/TeaOSLab/EdgeNode/internal/utils/kvstore"
	"github.com/iwind/TeaGo/assert"
	"runtime"
	"testing"
	"time"
)

func TestQuery_FindAll(t *testing.T) {
	var table = testOpenStoreTable[*testCachedItem](t, "cache_items", &testCacheItemEncoder[*testCachedItem]{})

	var before = time.Now()
	defer func() {
		t.Log("cost:", time.Since(before).Seconds()*1000, "ms")
	}()

	err := table.
		Query().
		Limit(10).
		//Offset("a1000").
		//Desc().
		FindAll(func(tx *kvstore.Tx[*testCachedItem], item kvstore.Item[*testCachedItem]) (goNext bool, err error) {
			t.Log("key:", item.Key, "value:", item.Value)

			return true, nil
		})
	if err != nil {
		t.Fatal(err)
	}
}

func TestQuery_FindAll_Break(t *testing.T) {
	var table = testOpenStoreTable[*testCachedItem](t, "cache_items", &testCacheItemEncoder[*testCachedItem]{})

	var before = time.Now()
	defer func() {
		t.Log("cost:", time.Since(before).Seconds()*1000, "ms")
	}()

	var count int
	err := table.
		Query().
		FindAll(func(tx *kvstore.Tx[*testCachedItem], item kvstore.Item[*testCachedItem]) (goNext bool, err error) {
			t.Log("key:", item.Key, "value:", item.Value)
			count++

			if count > 2 {
				// break test
				_ = table.DB().Store().Close()
			}

			return count < 3, nil
		})
	if err != nil {
		t.Fatal(err)
	}
}

func TestQuery_FindAll_Break_Closed(t *testing.T) {
	var table = testOpenStoreTable[*testCachedItem](t, "cache_items", &testCacheItemEncoder[*testCachedItem]{})

	var a = assert.NewAssertion(t)

	var before = time.Now()
	defer func() {
		t.Log("cost:", time.Since(before).Seconds()*1000, "ms")
	}()

	var count int
	err := table.
		Query().
		FindAll(func(tx *kvstore.Tx[*testCachedItem], item kvstore.Item[*testCachedItem]) (goNext bool, err error) {
			t.Log("key:", item.Key, "value:", item.Value)
			count++

			if count > 2 {
				// break test
				_ = table.DB().Store().Close()
			}

			return count < 3, nil
		})
	t.Log("expected error:", err)
	a.IsTrue(err != nil)
}

func TestQuery_FindAll_Desc(t *testing.T) {
	var table = testOpenStoreTable[*testCachedItem](t, "cache_items", &testCacheItemEncoder[*testCachedItem]{})

	err := table.Query().
		Desc().
		Limit(10).
		FindAll(func(tx *kvstore.Tx[*testCachedItem], item kvstore.Item[*testCachedItem]) (goNext bool, err error) {
			t.Log("key:", item.Key, "value:", item.Value)
			return true, nil
		})
	if err != nil {
		t.Fatal(err)
	}
}

func TestQuery_FindAll_Offset(t *testing.T) {
	var table = testOpenStoreTable[*testCachedItem](t, "cache_items", &testCacheItemEncoder[*testCachedItem]{})

	{
		t.Log("=== forward ===")
		err := table.Query().
			Offset("a3").
			Limit(10).
			FindAll(func(tx *kvstore.Tx[*testCachedItem], item kvstore.Item[*testCachedItem]) (goNext bool, err error) {
				t.Log("key:", item.Key, "value:", item.Value)
				return true, nil
			})
		if err != nil {
			t.Fatal(err)
		}
	}

	{
		t.Log("=== backward ===")
		err := table.Query().
			Desc().
			Offset("a3").
			Limit(10).
			//KeyOnly().
			FindAll(func(tx *kvstore.Tx[*testCachedItem], item kvstore.Item[*testCachedItem]) (goNext bool, err error) {
				t.Log("key:", item.Key, "value:", item.Value)
				return true, nil
			})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestQuery_FindAll_Count(t *testing.T) {
	var table = testOpenStoreTable[*testCachedItem](t, "cache_items", &testCacheItemEncoder[*testCachedItem]{})

	var count int

	var before = time.Now()
	defer func() {
		var costSeconds = time.Since(before).Seconds()
		t.Log("cost:", costSeconds*1000, "ms", "qps:", fmt.Sprintf("%.2fM/s", float64(count)/costSeconds/1_000_000))
	}()

	err := table.
		Query().
		KeysOnly().
		FindAll(func(tx *kvstore.Tx[*testCachedItem], item kvstore.Item[*testCachedItem]) (goNext bool, err error) {
			count++
			return true, nil
		})
	if err != nil {
		t.Fatal(err)
	}

	t.Log("count:", count)
}

func TestQuery_FindAll_Field(t *testing.T) {
	var table = testOpenStoreTable[*testCachedItem](t, "cache_items", &testCacheItemEncoder[*testCachedItem]{})

	var before = time.Now()
	defer func() {
		var costSeconds = time.Since(before).Seconds()
		t.Log("cost:", costSeconds*1000, "ms", "qps:", int(1/costSeconds))
	}()

	var lastFieldKey []byte

	t.Log("=======")
	{
		err := table.
			Query().
			FieldAsc("expiresAt").
			//KeysOnly().
			//FieldLt(1710848959).
			Limit(3).
			FindAll(func(tx *kvstore.Tx[*testCachedItem], item kvstore.Item[*testCachedItem]) (goNext bool, err error) {
				t.Log(item.Key, "=>", item.Value)
				lastFieldKey = item.FieldKey
				return true, nil
			})
		if err != nil {
			t.Fatal(err)
		}

	}

	t.Log("=======")
	{
		err := table.
			Query().
			FieldAsc("expiresAt").
			//KeysOnly().
			//FieldLt(1710848959).
			FieldOffset(lastFieldKey).
			Limit(3).
			FindAll(func(tx *kvstore.Tx[*testCachedItem], item kvstore.Item[*testCachedItem]) (goNext bool, err error) {
				t.Log(item.Key, "=>", item.Value)
				lastFieldKey = item.FieldKey
				return true, nil
			})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestQuery_FindAll_Field_Many(t *testing.T) {
	var table = testOpenStoreTable[*testCachedItem](t, "cache_items", &testCacheItemEncoder[*testCachedItem]{})

	var before = time.Now()
	defer func() {
		var costSeconds = time.Since(before).Seconds()
		t.Log("cost:", costSeconds*1000, "ms", "qps:", int(1/costSeconds))
	}()

	err := table.
		Query().
		FieldAsc("expiresAt").
		KeysOnly().
		Limit(1000).
		FindAll(func(tx *kvstore.Tx[*testCachedItem], item kvstore.Item[*testCachedItem]) (goNext bool, err error) {
			t.Log(item.Key, "=>", item.Value)
			return true, nil
		})
	if err != nil {
		t.Fatal(err)
	}
}

func BenchmarkQuery_FindAll(b *testing.B) {
	runtime.GOMAXPROCS(4)

	store, err := kvstore.OpenStore("test")
	if err != nil {
		b.Fatal(err)
	}
	defer func() {
		_ = store.Close()
	}()

	db, err := store.NewDB("db1")
	if err != nil {
		b.Fatal(err)
	}

	table, err := kvstore.NewTable[*testCachedItem]("cache_items", &testCacheItemEncoder[*testCachedItem]{})
	if err != nil {
		b.Fatal(err)
	}

	db.AddTable(table)

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err = table.Query().
				//Limit(100).
				FindAll(func(tx *kvstore.Tx[*testCachedItem], item kvstore.Item[*testCachedItem]) (goNext bool, err error) {
					return true, nil
				})
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
