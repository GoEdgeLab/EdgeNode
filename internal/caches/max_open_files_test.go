// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package caches_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/caches"
	"testing"
	"time"
)

func TestNewMaxOpenFiles(t *testing.T) {
	var maxOpenFiles = caches.NewMaxOpenFiles(2)
	maxOpenFiles.Fast()
	t.Log(maxOpenFiles.Max())

	maxOpenFiles.Fast()
	time.Sleep(1 * time.Second)
	t.Log(maxOpenFiles.Max())

	maxOpenFiles.Slow()
	t.Log(maxOpenFiles.Max())

	maxOpenFiles.Slow()
	t.Log(maxOpenFiles.Max())

	maxOpenFiles.Slow()
	t.Log(maxOpenFiles.Max())
}
