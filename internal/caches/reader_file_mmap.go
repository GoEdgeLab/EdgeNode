// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .
//go:build !plus

package caches

import (
	"errors"
	"io"
	"os"
)

func IsValidForMMAP(fp *os.File) (ok bool, stat os.FileInfo) {
	// stub
	return
}

type MMAPFileReader struct {
	FileReader
}

func NewMMAPFileReader(fp *os.File, stat os.FileInfo) (*MMAPFileReader, error) {
	// stub
	return &MMAPFileReader{}, errors.New("not implemented")
}

func (this *MMAPFileReader) CopyBodyTo(writer io.Writer) (int, error) {
	// stub
	return 0, errors.New("not implemented")
}
