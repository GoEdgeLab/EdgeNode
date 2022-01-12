// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package caches

import (
	"io"
	"os"
)

type OpenFile struct {
	fp      *os.File
	meta    []byte
	version int64
}

func NewOpenFile(fp *os.File, meta []byte) *OpenFile {
	return &OpenFile{
		fp:   fp,
		meta: meta,
	}
}

func (this *OpenFile) SeekStart() error {
	_, err := this.fp.Seek(0, io.SeekStart)
	return err
}

func (this *OpenFile) Close() error {
	this.meta = nil
	return this.fp.Close()
}
