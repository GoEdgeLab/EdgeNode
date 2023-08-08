// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package readers

import "io"

type FilterFunc = func(p []byte, readErr error) error

type FilterReaderCloser struct {
	rawReader io.Reader
	filters   []FilterFunc
}

func NewFilterReaderCloser(rawReader io.Reader) *FilterReaderCloser {
	return &FilterReaderCloser{
		rawReader: rawReader,
	}
}

func (this *FilterReaderCloser) Add(filter FilterFunc) {
	this.filters = append(this.filters, filter)
}

func (this *FilterReaderCloser) Read(p []byte) (n int, err error) {
	n, err = this.rawReader.Read(p)
	for _, filter := range this.filters {
		filterErr := filter(p[:n], err)
		if (err == nil || err != io.EOF) && filterErr != nil {
			err = filterErr
			return
		}
	}
	return
}

func (this *FilterReaderCloser) Close() error {
	closer, ok := this.rawReader.(io.Closer)
	if ok {
		return closer.Close()
	}
	return nil
}
