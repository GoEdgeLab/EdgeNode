// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package caches_test

import (
	"bytes"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/shared"
	"github.com/TeaOSLab/EdgeNode/internal/caches"
	"math/rand"
	"strconv"
	"testing"
	"time"
)

func TestNewMemoryWriter(t *testing.T) {
	var storage = caches.NewMemoryStorage(&serverconfigs.HTTPCachePolicy{
		Id:          0,
		IsOn:        false,
		Name:        "",
		Description: "",
		Capacity: &shared.SizeCapacity{
			Count: 8,
			Unit:  shared.SizeCapacityUnitGB,
		},
	}, nil)
	err := storage.Init()
	if err != nil {
		t.Fatal(err)
	}

	const size = 1 << 20
	const chunkSize = 16 << 10
	var data = bytes.Repeat([]byte{'A'}, chunkSize)

	var before = time.Now()

	var writer = caches.NewMemoryWriter(storage, "a", time.Now().Unix()+3600, 200, false, size, 1<<30, func(valueItem *caches.MemoryItem) {
		t.Log(len(valueItem.BodyValue), "bytes")
	})

	for i := 0; i < size/chunkSize; i++ {
		_, err = writer.Write(data)
		if err != nil {
			t.Fatal(err)
		}
	}

	err = writer.Close()
	if err != nil {
		t.Fatal(err)
	}

	t.Log("cost:", time.Since(before).Seconds()*1000, "ms")
}

func BenchmarkMemoryWriter_Capacity(b *testing.B) {
	b.ReportAllocs()

	var storage = caches.NewMemoryStorage(&serverconfigs.HTTPCachePolicy{
		Id:          0,
		IsOn:        false,
		Name:        "",
		Description: "",
		Capacity: &shared.SizeCapacity{
			Count: 8,
			Unit:  shared.SizeCapacityUnitGB,
		},
	}, nil)
	initErr := storage.Init()
	if initErr != nil {
		b.Fatal(initErr)
	}

	const size = 1 << 20
	const chunkSize = 16 << 10
	var data = bytes.Repeat([]byte{'A'}, chunkSize)

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var writer = caches.NewMemoryWriter(storage, "a"+strconv.Itoa(rand.Int()), time.Now().Unix()+3600, 200, false, size, 1<<30, func(valueItem *caches.MemoryItem) {
			})

			for i := 0; i < size/chunkSize; i++ {
				_, err := writer.Write(data)
				if err != nil {
					b.Fatal(err)
				}
			}

			err := writer.Close()
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkMemoryWriter_Capacity_Disabled(b *testing.B) {
	b.ReportAllocs()

	var storage = caches.NewMemoryStorage(&serverconfigs.HTTPCachePolicy{
		Id:          0,
		IsOn:        false,
		Name:        "",
		Description: "",
		Capacity: &shared.SizeCapacity{
			Count: 8,
			Unit:  shared.SizeCapacityUnitGB,
		},
	}, nil)
	initErr := storage.Init()
	if initErr != nil {
		b.Fatal(initErr)
	}

	const size = 1 << 20
	const chunkSize = 16 << 10
	var data = bytes.Repeat([]byte{'A'}, chunkSize)

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var writer = caches.NewMemoryWriter(storage, "a"+strconv.Itoa(rand.Int()), time.Now().Unix()+3600, 200, false, 0, 1<<30, func(valueItem *caches.MemoryItem) {
			})

			for i := 0; i < size/chunkSize; i++ {
				_, err := writer.Write(data)
				if err != nil {
					b.Fatal(err)
				}
			}

			err := writer.Close()
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
