// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package bfs

import "errors"

// FileWriter file writer
// not thread-safe
type FileWriter struct {
	bFile   *BlocksFile
	hasMeta bool
	hash    string

	bodySize     int64
	originOffset int64

	realHeaderSize int64
	realBodySize   int64
	isPartial      bool
}

func NewFileWriter(bFile *BlocksFile, hash string, bodySize int64, isPartial bool) (*FileWriter, error) {
	if isPartial && bodySize <= 0 {
		return nil, errors.New("invalid body size for partial content")
	}

	return &FileWriter{
		bFile:     bFile,
		hash:      hash,
		bodySize:  bodySize,
		isPartial: isPartial,
	}, nil
}

func (this *FileWriter) WriteMeta(status int, expiresAt int64, expectedFileSize int64) error {
	this.hasMeta = true
	return this.bFile.mFile.WriteMeta(this.hash, status, expiresAt, expectedFileSize)
}

func (this *FileWriter) WriteHeader(b []byte) (n int, err error) {
	if !this.isPartial && !this.hasMeta {
		err = errors.New("no meta found")
		return
	}

	n, err = this.bFile.Write(this.hash, BlockTypeHeader, b, -1)
	this.realHeaderSize += int64(n)
	return
}

func (this *FileWriter) WriteBody(b []byte) (n int, err error) {
	if !this.isPartial && !this.hasMeta {
		err = errors.New("no meta found")
		return
	}

	n, err = this.bFile.Write(this.hash, BlockTypeBody, b, this.originOffset)
	this.originOffset += int64(n)
	this.realBodySize += int64(n)
	return
}

func (this *FileWriter) WriteBodyAt(b []byte, offset int64) (n int, err error) {
	if !this.hasMeta {
		err = errors.New("no meta found")
		return
	}

	if !this.isPartial {
		err = errors.New("can not write body at specified offset: it is not a partial file")
		return
	}

	// still 'Write()' NOT 'WriteAt()'
	this.originOffset = offset
	n, err = this.bFile.Write(this.hash, BlockTypeBody, b, offset)
	this.originOffset += int64(n)
	return
}

func (this *FileWriter) Close() error {
	if !this.isPartial && !this.hasMeta {
		return errors.New("no meta found")
	}

	if this.isPartial {
		if this.originOffset > this.bodySize {
			return errors.New("unexpected body size")
		}
		this.realBodySize = this.bodySize
	} else {
		if this.bodySize > 0 && this.bodySize != this.realBodySize {
			return errors.New("unexpected body size")
		}
	}

	err := this.bFile.mFile.WriteClose(this.hash, this.realHeaderSize, this.realBodySize)
	if err != nil {
		return err
	}

	return this.bFile.Sync()
}

func (this *FileWriter) Discard() error {
	// TODO 需要测试
	return this.bFile.mFile.RemoveFile(this.hash)
}
