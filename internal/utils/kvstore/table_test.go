// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package kvstore_test

import (
	"fmt"
	"github.com/TeaOSLab/EdgeNode/internal/utils/kvstore"
	"github.com/TeaOSLab/EdgeNode/internal/utils/testutils"
	"github.com/iwind/TeaGo/assert"
	"github.com/iwind/TeaGo/types"
	"math/rand"
	"runtime"
	"strconv"
	"testing"
	"time"
)

func TestTable_Set(t *testing.T) {
	store, err := kvstore.OpenStore("test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = store.Close()
	}()

	db, err := store.NewDB("TEST_DB")
	if err != nil {
		t.Fatal(err)
	}

	table, err := kvstore.NewTable[string]("users", kvstore.NewStringValueEncoder[string]())
	if err != nil {
		t.Fatal(err)
	}

	db.AddTable(table)

	const originValue = "b12345"

	err = table.Set("a", originValue)
	if err != nil {
		t.Fatal(err)
	}

	value, err := table.Get("a")
	if err != nil {
		if kvstore.IsNotFound(err) {
			t.Log("not found key")
			return
		}
		t.Fatal(err)
	}
	t.Log("value:", value)

	var a = assert.NewAssertion(t)
	a.IsTrue(originValue == value)
}

func TestTable_Get(t *testing.T) {
	store, err := kvstore.OpenStore("test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = store.Close()
	}()

	db, err := store.NewDB("TEST_DB")
	if err != nil {
		t.Fatal(err)
	}

	table, err := kvstore.NewTable[string]("users", kvstore.NewStringValueEncoder[string]())
	if err != nil {
		t.Fatal(err)
	}

	db.AddTable(table)

	for _, key := range []string{"a", "b", "c"} {
		value, getErr := table.Get(key)
		if getErr != nil {
			if kvstore.IsNotFound(getErr) {
				t.Log("not found key", key)
				continue
			}
			t.Fatal(getErr)
		}
		t.Log(key, "=>", "value:", value)
	}
}

func TestTable_Exist(t *testing.T) {
	store, err := kvstore.OpenStore("test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = store.Close()
	}()

	db, err := store.NewDB("TEST_DB")
	if err != nil {
		t.Fatal(err)
	}

	table, err := kvstore.NewTable[string]("users", kvstore.NewStringValueEncoder[string]())
	if err != nil {
		t.Fatal(err)
	}

	db.AddTable(table)

	for _, key := range []string{"a", "b", "c", "12345"} {
		b, checkErr := table.Exist(key)
		if checkErr != nil {
			t.Fatal(checkErr)
		}
		t.Log(key, "=>", b)
	}
}

func TestTable_Delete(t *testing.T) {
	var a = assert.NewAssertion(t)

	store, err := kvstore.OpenStore("test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = store.Close()
	}()

	db, err := store.NewDB("TEST_DB")
	if err != nil {
		t.Fatal(err)
	}

	table, err := kvstore.NewTable[string]("users", kvstore.NewStringValueEncoder[string]())
	if err != nil {
		t.Fatal(err)
	}

	db.AddTable(table)

	value, err := table.Get("a123")
	if err != nil {
		if !kvstore.IsNotFound(err) {
			t.Fatal(err)
		}
	} else {
		t.Log("old value:", value)
	}

	err = table.Set("a123", "123456")
	if err != nil {
		t.Fatal(err)
	}

	{
		value, err = table.Get("a123")
		if err != nil {
			t.Fatal(err)
		}
		a.IsTrue(value == "123456")
	}

	err = table.Delete("a123")
	if err != nil {
		t.Fatal(err)
	}

	{
		_, err = table.Get("a123")
		a.IsTrue(kvstore.IsNotFound(err))
	}
}

func TestTable_Delete_Empty(t *testing.T) {
	var a = assert.NewAssertion(t)

	store, err := kvstore.OpenStore("test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = store.Close()
	}()

	db, err := store.NewDB("TEST_DB")
	if err != nil {
		t.Fatal(err)
	}

	table, err := kvstore.NewTable[string]("users", kvstore.NewStringValueEncoder[string]())
	if err != nil {
		t.Fatal(err)
	}

	db.AddTable(table)

	{
		err = table.Delete("a1", "a2", "a3", "a4", "")
		if err != nil {
			t.Fatal(err)
		}
	}

	{
		err = table.Delete()
		if err != nil {
			t.Fatal(err)
		}
	}

	// set new
	err = table.Set("a123", "123456")
	if err != nil {
		t.Fatal(err)
	}

	// delete again
	{
		err = table.Delete("a1", "a2", "a3", "a4", "")
		if err != nil {
			t.Fatal(err)
		}
	}

	{
		err = table.Delete()
		if err != nil {
			t.Fatal(err)
		}
	}

	// read
	{
		var value string
		value, err = table.Get("a123")
		if err != nil {
			t.Fatal(err)
		}
		a.IsTrue(value == "123456")
	}
}

func TestTable_Count(t *testing.T) {
	var table = testOpenStoreTable[*testCachedItem](t, "cache_items", &testCacheItemEncoder[*testCachedItem]{})

	var before = time.Now()
	count, err := table.Count()
	if err != nil {
		t.Fatal(err)
	}
	var costSeconds = time.Since(before).Seconds()
	t.Log("count:", count, "cost:", costSeconds*1000, "ms", "qps:", fmt.Sprintf("%.2fM/s", float64(count)/costSeconds/1_000_000))

	// watch memory usage
	if testutils.IsSingleTesting() {
		//time.Sleep(5 * time.Minute)
	}
}

func TestTable_Truncate(t *testing.T) {
	var table = testOpenStoreTable[*testCachedItem](t, "cache_items", &testCacheItemEncoder[*testCachedItem]{})
	var before = time.Now()
	err := table.Truncate()
	if err != nil {
		t.Fatal(err)
	}

	var costSeconds = time.Since(before).Seconds()
	t.Log("cost:", costSeconds*1000, "ms")

	t.Log("===after truncate===")
	testInspectDB(t)
}

func TestTable_ComposeFieldKey(t *testing.T) {
	var a = assert.NewAssertion(t)

	var table = testOpenStoreTable[*testCachedItem](t, "cache_items", &testCacheItemEncoder[*testCachedItem]{})
	var fieldKeyBytes = table.ComposeFieldKey([]byte("Lily"), "username", []byte("lucy"))
	t.Log(string(fieldKeyBytes))
	fieldValueBytes, keyValueBytes, err := table.DecodeFieldKey("username", fieldKeyBytes)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("field:", string(fieldValueBytes), "key:", string(keyValueBytes))
	a.IsTrue(string(fieldValueBytes) == "lucy")
	a.IsTrue(string(keyValueBytes) == "Lily")
}

func BenchmarkTable_Set(b *testing.B) {
	runtime.GOMAXPROCS(4)

	store, err := kvstore.OpenStore("test")
	if err != nil {
		b.Fatal(err)
	}
	defer func() {
		_ = store.Close()
	}()

	db, err := store.NewDB("TEST_DB")
	if err != nil {
		b.Fatal(err)
	}

	table, err := kvstore.NewTable[uint8]("users", kvstore.NewIntValueEncoder[uint8]())
	if err != nil {
		b.Fatal(err)
	}

	db.AddTable(table)

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			putErr := table.Set(strconv.Itoa(rand.Int()), 1)
			if putErr != nil {
				b.Fatal(putErr)
			}
		}
	})
}

func BenchmarkTable_Get(b *testing.B) {
	runtime.GOMAXPROCS(4)

	store, err := kvstore.OpenStore("test")
	if err != nil {
		b.Fatal(err)
	}
	defer func() {
		_ = store.Close()
	}()

	db, err := store.NewDB("TEST_DB")
	if err != nil {
		b.Fatal(err)
	}

	table, err := kvstore.NewTable[uint8]("users", kvstore.NewIntValueEncoder[uint8]())
	if err != nil {
		b.Fatal(err)
	}

	db.AddTable(table)

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, putErr := table.Get(types.String(rand.Int()))
			if putErr != nil {
				if kvstore.IsNotFound(putErr) {
					continue
				}
				b.Fatal(putErr)
			}
		}
	})
}

/**func BenchmarkTable_NextId(b *testing.B) {
	runtime.GOMAXPROCS(4)

	store, err := kvstore.OpenStore("test")
	if err != nil {
		b.Fatal(err)
	}
	defer func() {
		_ = store.Close()
	}()

	db, err := store.NewDB("TEST_DB")
	if err != nil {
		b.Fatal(err)
	}

	table, err := kvstore.NewTable[uint8]("users", kvstore.NewIntValueEncoder[uint8]())
	if err != nil {
		b.Fatal(err)
	}

	db.AddTable(table)

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, nextErr := table.NextId("a")
			if nextErr != nil {
				b.Fatal(nextErr)
			}
		}
	})
}
**/
