// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package kvstore_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils/kvstore"
	"runtime"
	"testing"
)

func TestCounterTable_Increase(t *testing.T) {
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

	count, err := table.Increase("counter", 1)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(count)
}

func BenchmarkCounterTable_Increase(b *testing.B) {
	runtime.GOMAXPROCS(1)

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

	table, err := kvstore.NewCounterTable[uint64]("users_counter")
	if err != nil {
		b.Fatal(err)
	}

	db.AddTable(table)

	defer func() {
		count, incrErr := table.Increase("counter", 1)
		if incrErr != nil {
			b.Fatal(incrErr)
		}
		b.Log(count)
	}()

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, incrErr := table.Increase("counter", 1)
			if incrErr != nil {
				b.Fatal(incrErr)
			}
		}
	})
}
