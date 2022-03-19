// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package compressions

type BaseReader struct {
	pool *ReaderPool

	isFinished bool
}

func (this *BaseReader) SetPool(pool *ReaderPool) {
	this.pool = pool
}

func (this *BaseReader) Finish(obj Reader) error {
	err := obj.RawClose()
	if err == nil && this.pool != nil && !this.isFinished {
		this.pool.Put(obj)
	}
	this.isFinished = true
	return err
}

func (this *BaseReader) ResetFinish() {
	this.isFinished = false
}
