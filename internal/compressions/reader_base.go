// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package compressions

import "sync/atomic"

type BaseReader struct {
	pool *ReaderPool

	isFinished bool
	hits       uint32
}

func (this *BaseReader) SetPool(pool *ReaderPool) {
	this.pool = pool
}

func (this *BaseReader) Finish(obj Reader) error {
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

func (this *BaseReader) ResetFinish() {
	this.isFinished = false
}

func (this *BaseReader) IncreaseHit() uint32 {
	return atomic.AddUint32(&this.hits, 1)
}
