// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package utils_test

import (
	"bytes"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"strings"
	"testing"
)

func TestNewBufferPool(t *testing.T) {
	var pool = utils.NewBufferPool()
	var b = pool.Get()
	b.WriteString("Hello, World")
	t.Log(b.String())

	pool.Put(b)
	t.Log(b.String())

	b = pool.Get()
	t.Log(b.String())
}

func BenchmarkNewBufferPool1(b *testing.B) {
	var data = []byte(strings.Repeat("Hello", 1024))

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var buffer = &bytes.Buffer{}
			buffer.Write(data)
		}
	})
}

func BenchmarkNewBufferPool2(b *testing.B) {
	var pool = utils.NewBufferPool()
	var data = []byte(strings.Repeat("Hello", 1024))

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var buffer = pool.Get()
			buffer.Write(data)
			pool.Put(buffer)
		}
	})
}
