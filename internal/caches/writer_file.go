package caches

import (
	"encoding/binary"
	"github.com/iwind/TeaGo/types"
	"io"
	"os"
	"strings"
)

type FileWriter struct {
	rawWriter  *os.File
	key        string
	headerSize int64
	bodySize   int64
	expiredAt  int64
	endFunc    func()
}

func NewFileWriter(rawWriter *os.File, key string, expiredAt int64, endFunc func()) *FileWriter {
	return &FileWriter{
		key:       key,
		rawWriter: rawWriter,
		expiredAt: expiredAt,
		endFunc:   endFunc,
	}
}

// WriteHeader 写入数据
func (this *FileWriter) WriteHeader(data []byte) (n int, err error) {
	n, err = this.rawWriter.Write(data)
	this.headerSize += int64(n)
	if err != nil {
		_ = this.Discard()
	}
	return
}

// WriteHeaderLength 写入Header长度数据
func (this *FileWriter) WriteHeaderLength(headerLength int) error {
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
func (this *FileWriter) Write(data []byte) (n int, err error) {
	n, err = this.rawWriter.Write(data)
	this.bodySize += int64(n)
	if err != nil {
		_ = this.Discard()
	}
	return
}

// WriteBodyLength 写入Body长度数据
func (this *FileWriter) WriteBodyLength(bodyLength int64) error {
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
func (this *FileWriter) Close() error {
	defer this.endFunc()

	err := this.WriteHeaderLength(types.Int(this.headerSize))
	if err != nil {
		return err
	}
	err = this.WriteBodyLength(this.bodySize)
	if err != nil {
		return err
	}

	path := this.rawWriter.Name()
	err = this.rawWriter.Close()
	if err != nil {
		_ = os.Remove(path)
	} else {
		err = os.Rename(path, strings.Replace(path, ".tmp", "", 1))
		if err != nil {
			_ = os.Remove(path)
		}
	}

	return err
}

// Discard 丢弃
func (this *FileWriter) Discard() error {
	defer this.endFunc()

	_ = this.rawWriter.Close()

	err := os.Remove(this.rawWriter.Name())
	return err
}

func (this *FileWriter) HeaderSize() int64 {
	return this.headerSize
}

func (this *FileWriter) BodySize() int64 {
	return this.bodySize
}

func (this *FileWriter) ExpiredAt() int64 {
	return this.expiredAt
}

func (this *FileWriter) Key() string {
	return this.key
}

// ItemType 获取内容类型
func (this *FileWriter) ItemType() ItemType {
	return ItemTypeFile
}
