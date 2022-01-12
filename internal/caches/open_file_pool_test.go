// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package caches_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/caches"
	"testing"
)

func TestOpenFilePool_Get(t *testing.T) {
	var pool = caches.NewOpenFilePool("a")
	t.Log(pool.Filename())
	t.Log(pool.Get())
	t.Log(pool.Put(caches.NewOpenFile(nil, nil)))
	t.Log(pool.Get())
	t.Log(pool.Get())
}
