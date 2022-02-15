// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package readers

import (
	"io"
)

type TeeReader struct {
	r io.Reader
	w io.Writer

	onFail func(err error)
	onEOF  func()
}

func NewTeeReader(reader io.Reader, writer io.Writer) *TeeReader {
	return &TeeReader{
		r: reader,
		w: writer,
	}
}

func (this *TeeReader) Read(p []byte) (n int, err error) {
	n, err = this.r.Read(p)
	if n > 0 {
		_, wErr := this.w.Write(p[:n])
		if err == nil && wErr != nil {
			err = wErr
		}
	}
	if err != nil {
		if err == io.EOF {
			if this.onEOF != nil {
				this.onEOF()
			}
		} else {
			if this.onFail != nil {
				this.onFail(err)
			}
		}
	}
	return
}

func (this *TeeReader) OnFail(onFail func(err error)) {
	this.onFail = onFail
}

func (this *TeeReader) OnEOF(onEOF func()) {
	this.onEOF = onEOF
}
