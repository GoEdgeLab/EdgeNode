// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package compressions

import (
	"io"
)

type WriterPool struct {
	m       map[int]chan Writer // level => chan Writer
	newFunc func(writer io.Writer, level int) (Writer, error)
}

func NewWriterPool(maxSize int, maxLevel int, newFunc func(writer io.Writer, level int) (Writer, error)) *WriterPool {
	if maxSize <= 0 {
		maxSize = 1024
	}

	var m = map[int]chan Writer{}
	for i := 0; i <= maxLevel; i++ {
		m[i] = make(chan Writer, maxSize)
	}

	return &WriterPool{
		m:       m,
		newFunc: newFunc,
	}
}

func (this *WriterPool) Get(parentWriter io.Writer, level int) (Writer, error) {
	c, ok := this.m[level]
	if !ok {
		c = this.m[0]
	}

	select {
	case writer := <-c:
		writer.Reset(parentWriter)
		writer.ResetFinish()
		return writer, nil
	default:
		writer, err := this.newFunc(parentWriter, level)
		if err != nil {
			return nil, err
		}
		writer.SetPool(this)
		return writer, nil
	}
}

func (this *WriterPool) Put(writer Writer) {
	var level = writer.Level()
	c, ok := this.m[level]
	if !ok {
		c = this.m[0]
	}
	select {
	case c <- writer:
	default:
	}
}
