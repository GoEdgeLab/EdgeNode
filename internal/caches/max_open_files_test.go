// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package caches_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/caches"
	"testing"
	"time"
)

func TestNewMaxOpenFiles(t *testing.T) {
	var maxOpenFiles = caches.NewMaxOpenFiles()
	maxOpenFiles.Fast()
	t.Log("fast:", maxOpenFiles.Next())

	maxOpenFiles.Slow()
	t.Log("slow:", maxOpenFiles.Next())
	time.Sleep(1*time.Second + 1*time.Millisecond)
	t.Log("slow 1 second:", maxOpenFiles.Next())

	maxOpenFiles.Slow()
	t.Log("slow:", maxOpenFiles.Next())

	maxOpenFiles.Slow()
	t.Log("slow:", maxOpenFiles.Next())

	time.Sleep(1 * time.Second)
	t.Log("slow 1 second:", maxOpenFiles.Next())

	maxOpenFiles.Slow()
	t.Log("slow:", maxOpenFiles.Next())

	maxOpenFiles.Fast()
	t.Log("fast:", maxOpenFiles.Next())
}
