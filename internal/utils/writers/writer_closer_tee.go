// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package writers

import (
	"io"
)

type TeeWriterCloser struct {
	primaryW   io.WriteCloser
	secondaryW io.WriteCloser

	onFail func(err error)
}

func NewTeeWriterCloser(primaryW io.WriteCloser, secondaryW io.WriteCloser) *TeeWriterCloser {
	return &TeeWriterCloser{
		primaryW:   primaryW,
		secondaryW: secondaryW,
	}
}

func (this *TeeWriterCloser) Write(p []byte) (n int, err error) {
	{
		n, err = this.primaryW.Write(p)

		if err != nil {
			if this.onFail != nil {
				this.onFail(err)
			}
		}
	}

	{
		_, err2 := this.secondaryW.Write(p)
		if err2 != nil {
			if this.onFail != nil {
				this.onFail(err2)
			}
		}
	}

	return
}

func (this *TeeWriterCloser) Close() error {
	// 这里不关闭secondary
	return this.primaryW.Close()
}

func (this *TeeWriterCloser) OnFail(onFail func(err error)) {
	this.onFail = onFail
}
