// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package compressions

import (
	"compress/gzip"
	"io"
)

type GzipReader struct {
	reader *gzip.Reader
}

func NewGzipReader(reader io.Reader) (Reader, error) {
	r, err := gzip.NewReader(reader)
	if err != nil {
		return nil, err
	}
	return &GzipReader{
		reader: r,
	}, nil
}

func (this *GzipReader) Read(p []byte) (n int, err error) {
	return this.reader.Read(p)
}

func (this *GzipReader) Close() error {
	return this.reader.Close()
}
