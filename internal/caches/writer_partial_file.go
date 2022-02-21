// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package caches

import (
	"encoding/binary"
	"github.com/iwind/TeaGo/types"
	"io"
	"os"
	"strings"
	"sync"
)

type PartialFileWriter struct {
	rawWriter  *os.File
	key        string
	headerSize int64
	bodySize   int64
	expiredAt  int64
	endFunc    func()
	once       sync.Once

	isNew      bool
	isPartial  bool
	bodyOffset int64
}

func NewPartialFileWriter(rawWriter *os.File, key string, expiredAt int64, isNew bool, isPartial bool, bodyOffset int64, endFunc func()) *PartialFileWriter {
	return &PartialFileWriter{
		key:        key,
		rawWriter:  rawWriter,
		expiredAt:  expiredAt,
		endFunc:    endFunc,
		isNew:      isNew,
		isPartial:  isPartial,
		bodyOffset: bodyOffset,
	}
}

// WriteHeader 写入数据
func (this *PartialFileWriter) WriteHeader(data []byte) (n int, err error) {
	if !this.isNew {
		return
	}
	n, err = this.rawWriter.Write(data)
	this.headerSize += int64(n)
	if err != nil {
		_ = this.Discard()
	}
	return
}

// WriteHeaderLength 写入Header长度数据
func (this *PartialFileWriter) WriteHeaderLength(headerLength int) error {
	bytes4 := make([]byte, 4)
	binary.BigEndian.PutUint32(bytes4, uint32(headerLength))
	_, err := this.rawWriter.Seek(SizeExpiresAt+SizeStatus+SizeURLLength, io.SeekStart)
	if err != nil {
		_ = this.Discard()
		return err
	}
	_, err = this.rawWriter.Write(bytes4)
	if err != nil {
		_ = this.Discard()
		return err
	}
	return nil
}

// Write 写入数据
func (this *PartialFileWriter) Write(data []byte) (n int, err error) {
	n, err = this.rawWriter.Write(data)
	this.bodySize += int64(n)
	if err != nil {
		_ = this.Discard()
	}
	return
}

// WriteAt 在指定位置写入数据
func (this *PartialFileWriter) WriteAt(data []byte, offset int64) error {
	if this.bodyOffset == 0 {
		this.bodyOffset = SizeMeta + int64(len(this.key)) + this.headerSize
	}
	_, err := this.rawWriter.WriteAt(data, this.bodyOffset+offset)
	return err
}

// WriteBodyLength 写入Body长度数据
func (this *PartialFileWriter) WriteBodyLength(bodyLength int64) error {
	bytes8 := make([]byte, 8)
	binary.BigEndian.PutUint64(bytes8, uint64(bodyLength))
	_, err := this.rawWriter.Seek(SizeExpiresAt+SizeStatus+SizeURLLength+SizeHeaderLength, io.SeekStart)
	if err != nil {
		_ = this.Discard()
		return err
	}
	_, err = this.rawWriter.Write(bytes8)
	if err != nil {
		_ = this.Discard()
		return err
	}
	return nil
}

// Close 关闭
func (this *PartialFileWriter) Close() error {
	defer this.once.Do(func() {
		this.endFunc()
	})

	var path = this.rawWriter.Name()

	if this.isNew {
		err := this.WriteHeaderLength(types.Int(this.headerSize))
		if err != nil {
			_ = this.rawWriter.Close()
			_ = os.Remove(path)
			return err
		}
		err = this.WriteBodyLength(this.bodySize)
		if err != nil {
			_ = this.rawWriter.Close()
			_ = os.Remove(path)
			return err
		}
	}

	err := this.rawWriter.Close()
	if err != nil {
		_ = os.Remove(path)
	} else if !this.isPartial {
		err = os.Rename(path, strings.Replace(path, ".tmp", "", 1))
		if err != nil {
			_ = os.Remove(path)
		}
	}

	return err
}

// Discard 丢弃
func (this *PartialFileWriter) Discard() error {
	defer this.once.Do(func() {
		this.endFunc()
	})

	_ = this.rawWriter.Close()

	err := os.Remove(this.rawWriter.Name())
	return err
}

func (this *PartialFileWriter) HeaderSize() int64 {
	return this.headerSize
}

func (this *PartialFileWriter) BodySize() int64 {
	return this.bodySize
}

func (this *PartialFileWriter) ExpiredAt() int64 {
	return this.expiredAt
}

func (this *PartialFileWriter) Key() string {
	return this.key
}

// ItemType 获取内容类型
func (this *PartialFileWriter) ItemType() ItemType {
	return ItemTypeFile
}
