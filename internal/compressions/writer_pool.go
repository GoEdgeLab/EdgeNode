// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package compressions

import (
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"io"
	"time"
)

const maxWriterHits = 1 << 20

var isBusy = false

func init() {
	if !teaconst.IsMain {
		return
	}

	goman.New(func() {
		var ticker = time.NewTicker(100 * time.Millisecond)
		for range ticker.C {
			if isBusy {
				isBusy = false
			}
		}
	})
}

func IsBusy() bool {
	return isBusy
}

type WriterPool struct {
	c       chan Writer // level => chan Writer
	newFunc func(writer io.Writer, level int) (Writer, error)
}

func NewWriterPool(maxSize int, newFunc func(writer io.Writer, level int) (Writer, error)) *WriterPool {
	if maxSize <= 0 {
		maxSize = 1024
	}

	return &WriterPool{
		c:       make(chan Writer, maxSize),
		newFunc: newFunc,
	}
}

func (this *WriterPool) Get(parentWriter io.Writer, level int) (Writer, error) {
	if isBusy {
		return nil, ErrIsBusy
	}

	select {
	case writer := <-this.c:
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
	if writer.IncreaseHit() > maxWriterHits {
		// do nothing to discard it
		return
	}

	select {
	case this.c <- writer:
	default:
		isBusy = true
	}
}
