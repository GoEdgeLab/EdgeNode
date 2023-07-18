// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package readers

import (
	"io"
)

type TeeReaderCloser struct {
	r io.Reader
	w io.Writer

	onFail func(err error)
	onEOF  func()
}

func NewTeeReaderCloser(reader io.Reader, writer io.Writer) *TeeReaderCloser {
	return &TeeReaderCloser{
		r: reader,
		w: writer,
	}
}

func (this *TeeReaderCloser) Read(p []byte) (n int, err error) {
	n, err = this.r.Read(p)
	if n > 0 {
		_, wErr := this.w.Write(p[:n])
		if (err == nil || err == io.EOF) && wErr != nil {
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

func (this *TeeReaderCloser) Close() error {
	r, ok := this.r.(io.Closer)
	if ok {
		return r.Close()
	}
	return nil
}

func (this *TeeReaderCloser) OnFail(onFail func(err error)) {
	this.onFail = onFail
}

func (this *TeeReaderCloser) OnEOF(onEOF func()) {
	this.onEOF = onEOF
}
