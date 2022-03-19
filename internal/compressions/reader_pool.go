// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package compressions

import (
	"io"
)

type ReaderPool struct {
	c       chan Reader
	newFunc func(reader io.Reader) (Reader, error)
}

func NewReaderPool(maxSize int, newFunc func(reader io.Reader) (Reader, error)) *ReaderPool {
	if maxSize <= 0 {
		maxSize = 1024
	}

	return &ReaderPool{
		c:       make(chan Reader, maxSize),
		newFunc: newFunc,
	}
}

func (this *ReaderPool) Get(parentReader io.Reader) (Reader, error) {
	select {
	case reader := <-this.c:
		err := reader.Reset(parentReader)
		if err != nil {
			// create new
			reader, err = this.newFunc(parentReader)
			if err != nil {
				return nil, err
			}
			reader.SetPool(this)
			return reader, nil
		}
		reader.ResetFinish()
		return reader, nil
	default:
		// create new
		reader, err := this.newFunc(parentReader)
		if err != nil {
			return nil, err
		}
		reader.SetPool(this)
		return reader, nil
	}
}

func (this *ReaderPool) Put(reader Reader) {
	select {
	case this.c <- reader:
	default:
	}
}
