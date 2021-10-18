// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package compressions

import (
	"compress/flate"
	"io"
)

type DeflateReader struct {
	reader io.ReadCloser
}

func NewDeflateReader(reader io.Reader) (Reader, error) {
	return &DeflateReader{reader: flate.NewReader(reader)}, nil
}

func (this *DeflateReader) Read(p []byte) (n int, err error) {
	return this.reader.Read(p)
}

func (this *DeflateReader) Close() error {
	return this.reader.Close()
}
