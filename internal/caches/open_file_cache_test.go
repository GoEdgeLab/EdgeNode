// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package caches_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/caches"
	"github.com/TeaOSLab/EdgeNode/internal/utils/testutils"
	"testing"
	"time"
)

func TestNewOpenFileCache_Close(t *testing.T) {
	cache, err := caches.NewOpenFileCache(1024)
	if err != nil {
		t.Fatal(err)
	}
	cache.Debug()
	cache.Put("a.txt", caches.NewOpenFile(nil, nil, nil, 0))
	cache.Put("b.txt", caches.NewOpenFile(nil, nil, nil, 0))
	cache.Put("b.txt", caches.NewOpenFile(nil, nil, nil, 0))
	cache.Put("b.txt", caches.NewOpenFile(nil, nil, nil, 0))
	cache.Put("c.txt", caches.NewOpenFile(nil, nil, nil, 0))
	cache.Get("b.txt")
	cache.Get("d.txt")
	cache.Close("a.txt")

	if testutils.IsSingleTesting() {
		time.Sleep(100 * time.Second)
	}
}

func TestNewOpenFileCache_CloseAll(t *testing.T) {
	cache, err := caches.NewOpenFileCache(1024)
	if err != nil {
		t.Fatal(err)
	}
	cache.Debug()
	cache.Put("a.txt", caches.NewOpenFile(nil, nil, nil, 0))
	cache.Put("b.txt", caches.NewOpenFile(nil, nil, nil, 0))
	cache.Put("c.txt", caches.NewOpenFile(nil, nil, nil, 0))
	cache.Get("b.txt")
	cache.Get("d.txt")
	cache.CloseAll()

	time.Sleep(6 * time.Second)
}
