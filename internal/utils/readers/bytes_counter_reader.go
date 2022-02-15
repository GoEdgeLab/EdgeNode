// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package readers

import "io"

type BytesCounterReader struct {
	rawReader io.Reader
	count     int64
}

func NewBytesCounterReader(rawReader io.Reader) *BytesCounterReader {
	return &BytesCounterReader{
		rawReader: rawReader,
	}
}

func (this *BytesCounterReader) Read(p []byte) (n int, err error) {
	n, err = this.rawReader.Read(p)
	this.count += int64(n)
	return
}

func (this *BytesCounterReader) TotalBytes() int64 {
	return this.count
}
