// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package compressions

import (
	"sync/atomic"
)

type BaseWriter struct {
	pool *WriterPool

	isFinished bool

	hits uint32
}

func (this *BaseWriter) SetPool(pool *WriterPool) {
	this.pool = pool
}

func (this *BaseWriter) Finish(obj Writer) error {
	if this.isFinished {
		return nil
	}
	err := obj.RawClose()
	if err == nil && this.pool != nil {
		this.pool.Put(obj)
	}
	this.isFinished = true
	return err
}

func (this *BaseWriter) ResetFinish() {
	this.isFinished = false
}

func (this *BaseWriter) IncreaseHit() uint32 {
	return atomic.AddUint32(&this.hits, 1)
}
