// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package readers

import "io"

type FilterFunc = func(p []byte, err error) error

type FilterReader struct {
	rawReader io.Reader
	filters   []FilterFunc
}

func NewFilterReader(rawReader io.Reader) *FilterReader {
	return &FilterReader{
		rawReader: rawReader,
	}
}

func (this *FilterReader) Add(filter FilterFunc) {
	this.filters = append(this.filters, filter)
}

func (this *FilterReader) Read(p []byte) (n int, err error) {
	n, err = this.rawReader.Read(p)
	for _, filter := range this.filters {
		filterErr := filter(p[:n], err)
		if filterErr != nil {
			err = filterErr
			return
		}
	}
	return
}
