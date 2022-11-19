// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package utils

import (
	"bytes"
	"sync"
)

var SharedBufferPool = NewBufferPool()

// BufferPool pool for get byte slice
type BufferPool struct {
	rawPool *sync.Pool
}

// NewBufferPool 创建新对象
func NewBufferPool() *BufferPool {
	var pool = &BufferPool{}
	pool.rawPool = &sync.Pool{
		New: func() any {
			return &bytes.Buffer{}
		},
	}
	return pool
}

// Get 获取一个新的Buffer
func (this *BufferPool) Get() (b *bytes.Buffer) {
	var buffer = this.rawPool.Get().(*bytes.Buffer)
	if buffer.Len() > 0 {
		buffer.Reset()
	}
	return buffer
}

// Put 放回一个使用过的byte slice
func (this *BufferPool) Put(b *bytes.Buffer) {
	this.rawPool.Put(b)
}
