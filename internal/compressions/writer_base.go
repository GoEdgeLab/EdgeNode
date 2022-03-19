// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package compressions

type BaseWriter struct {
	pool *WriterPool

	isFinished bool
}

func (this *BaseWriter) SetPool(pool *WriterPool) {
	this.pool = pool
}

func (this *BaseWriter) Finish(obj Writer) error {
	err := obj.RawClose()
	if err == nil && this.pool != nil && !this.isFinished {
		this.pool.Put(obj)
	}
	this.isFinished = true
	return err
}

func (this *BaseWriter) ResetFinish() {
	this.isFinished = false
}
