// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package bfs

import (
	"bytes"
	"encoding/json"
	"github.com/klauspost/compress/gzip"
)

// LazyFileHeader load file header lazily to save memory
type LazyFileHeader struct {
	rawData    []byte
	fileHeader *FileHeader
}

func NewLazyFileHeaderFromData(rawData []byte) *LazyFileHeader {
	return &LazyFileHeader{
		rawData: rawData,
	}
}

func NewLazyFileHeader(fileHeader *FileHeader) *LazyFileHeader {
	return &LazyFileHeader{
		fileHeader: fileHeader,
	}
}

func (this *LazyFileHeader) FileHeaderUnsafe() (*FileHeader, error) {
	if this.fileHeader != nil {
		return this.fileHeader, nil
	}

	// TODO 使用pool管理gzip
	gzReader, err := gzip.NewReader(bytes.NewBuffer(this.rawData))
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = gzReader.Close()
	}()

	var header = &FileHeader{}
	err = json.NewDecoder(gzReader).Decode(header)
	if err != nil {
		return nil, err
	}

	header.IsWriting = false

	this.fileHeader = header
	this.rawData = nil

	return header, nil
}
