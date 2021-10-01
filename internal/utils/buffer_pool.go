// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package utils

import "bytes"

// BufferPool pool for get byte slice
type BufferPool struct {
	c chan *bytes.Buffer
}

// NewBufferPool 创建新对象
func NewBufferPool(maxSize int) *BufferPool {
	if maxSize <= 0 {
		maxSize = 1024
	}
	pool := &BufferPool{
		c: make(chan *bytes.Buffer, maxSize),
	}
	return pool
}

// Get 获取一个新的Buffer
func (this *BufferPool) Get() (b *bytes.Buffer) {
	select {
	case b = <-this.c:
		b.Reset()
	default:
		b = &bytes.Buffer{}
	}
	return
}

// Put 放回一个使用过的byte slice
func (this *BufferPool) Put(b *bytes.Buffer) {
	b.Reset()

	select {
	case this.c <- b:
	default:
		// 已达最大容量，则抛弃
	}
}

// Size 当前的数量
func (this *BufferPool) Size() int {
	return len(this.c)
}
