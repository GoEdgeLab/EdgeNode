// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package compressions

import (
	"compress/flate"
	"io"
)

type DeflateReader struct {
	BaseReader

	reader io.ReadCloser
}

func NewDeflateReader(reader io.Reader) (Reader, error) {
	return sharedDeflateReaderPool.Get(reader)
}

func newDeflateReader(reader io.Reader) (Reader, error) {
	return &DeflateReader{reader: flate.NewReader(reader)}, nil
}

func (this *DeflateReader) Read(p []byte) (n int, err error) {
	return this.reader.Read(p)
}

func (this *DeflateReader) Reset(reader io.Reader) error {
	this.reader = flate.NewReader(reader)
	return nil
}

func (this *DeflateReader) RawClose() error {
	return this.reader.Close()
}

func (this *DeflateReader) Close() error {
	return this.Finish(this)
}
