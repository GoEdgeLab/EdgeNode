// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package kvstore_test

import (
	"fmt"
	"github.com/TeaOSLab/EdgeNode/internal/utils/kvstore"
	"testing"
)

func TestTable_ReadTx(t *testing.T) {
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

	table, err := kvstore.NewCounterTable[uint64]("users_counter")
	if err != nil {
		t.Fatal(err)
	}

	db.AddTable(table)

	err = table.WriteTx(func(tx *kvstore.Tx[uint64]) error {
		for i := 0; i < 1000; i++ {
			var key = fmt.Sprintf("a%03d", i)
			setErr := tx.Set(key, uint64(i))
			if setErr != nil {
				return setErr
			}

			value, getErr := tx.Get(key)
			if getErr != nil {
				return getErr
			}
			t.Log("write:", key, "=>", value)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	err = table.ReadTx(func(tx *kvstore.Tx[uint64]) error {
		for _, key := range []string{"a100", "a101", "a102"} {
			value, getErr := tx.Get(key)
			if getErr != nil {
				return getErr
			}
			t.Log("read:", key, "=>", value)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
