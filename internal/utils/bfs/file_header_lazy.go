// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package bfs

import (
	"bytes"
	"encoding/json"
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

	var jsonPrefix = []byte("json:")

	var header = &FileHeader{}

	// json
	if bytes.HasPrefix(this.rawData, jsonPrefix) {
		err := json.Unmarshal(this.rawData[len(jsonPrefix):], header)
		if err != nil {
			return nil, err
		}
		return header, nil
	}

	decompressor, err := SharedDecompressPool.Get(bytes.NewBuffer(this.rawData))
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = decompressor.Close()
		SharedDecompressPool.Put(decompressor)
	}()

	err = json.NewDecoder(decompressor).Decode(header)
	if err != nil {
		return nil, err
	}

	header.IsWriting = false

	this.fileHeader = header
	this.rawData = nil

	return header, nil
}
