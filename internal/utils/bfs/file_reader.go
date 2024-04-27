// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package bfs

import (
	"errors"
	"github.com/iwind/TeaGo/types"
	"io"
	"os"
)

type FileReader struct {
	bFile *BlocksFile
	fp    *os.File

	fileHeader *FileHeader
	pos        int64

	isClosed bool
}

func NewFileReader(bFile *BlocksFile, fp *os.File, fileHeader *FileHeader) *FileReader {
	return &FileReader{
		bFile:      bFile,
		fp:         fp,
		fileHeader: fileHeader,
	}
}

func (this *FileReader) FileHeader() *FileHeader {
	return this.fileHeader
}

func (this *FileReader) Read(b []byte) (n int, err error) {
	n, err = this.ReadAt(b, this.pos)
	this.pos += int64(n)

	return
}

func (this *FileReader) ReadAt(b []byte, offset int64) (n int, err error) {
	if offset >= this.fileHeader.MaxOffset() {
		err = io.EOF
		return
	}

	blockInfo, ok := this.fileHeader.BlockAt(offset)
	if !ok {
		err = errors.New("could not find block at '" + types.String(offset) + "'")
		return
	}

	var delta = offset - blockInfo.OriginOffsetFrom
	var bFrom = blockInfo.BFileOffsetFrom + delta
	var bTo = blockInfo.BFileOffsetTo
	if bFrom > bTo {
		err = errors.New("invalid block information")
		return
	}

	var bufLen = len(b)
	if int64(bufLen) > bTo-bFrom {
		bufLen = int(bTo - bFrom)
	}

	AckReadThread()
	n, err = this.fp.ReadAt(b[:bufLen], bFrom)
	ReleaseReadThread()

	return
}

func (this *FileReader) Reset(fileHeader *FileHeader) {
	this.fileHeader = fileHeader
	this.pos = 0
}

func (this *FileReader) Close() error {
	if this.isClosed {
		return nil
	}
	this.isClosed = true
	return this.bFile.CloseFileReader(this)
}

func (this *FileReader) Free() error {
	return this.fp.Close()
}
