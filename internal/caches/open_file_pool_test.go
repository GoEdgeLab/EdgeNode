// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package caches_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/caches"
	"github.com/iwind/TeaGo/rands"
	"sync"
	"testing"
)

func TestOpenFilePool_Get(t *testing.T) {
	var pool = caches.NewOpenFilePool("a")
	t.Log(pool.Filename())
	t.Log(pool.Get())
	t.Log(pool.Put(caches.NewOpenFile(nil, nil, nil, 0, 1)))
	t.Log(pool.Get())
	t.Log(pool.Get())
}

func TestOpenFilePool_Close(t *testing.T) {
	var pool = caches.NewOpenFilePool("a")
	pool.Put(caches.NewOpenFile(nil, nil, nil, 0, 1))
	pool.Put(caches.NewOpenFile(nil, nil, nil, 0, 1))
	pool.Close()
}

func TestOpenFilePool_Concurrent(t *testing.T) {
	var pool = caches.NewOpenFilePool("a")
	var concurrent = 1000
	var wg = &sync.WaitGroup{}
	wg.Add(concurrent)
	for i := 0; i < concurrent; i++ {
		go func() {
			defer wg.Done()

			if rands.Int(0, 1) == 1 {
				pool.Put(caches.NewOpenFile(nil, nil, nil, 0, 1))
			}
			if rands.Int(0, 1) == 0 {
				pool.Get()
			}
		}()
	}
	wg.Wait()
}
