package caches

import (
	"github.com/TeaOSLab/EdgeNode/internal/compressions"
)

type compressionWriter struct {
	rawWriter Writer
	writer    compressions.Writer
	key       string
	expiredAt int64
}

func NewCompressionWriter(gw Writer, cpWriter compressions.Writer, key string, expiredAt int64) Writer {
	return &compressionWriter{
		rawWriter: gw,
		writer:    cpWriter,
		key:       key,
		expiredAt: expiredAt,
	}
}

func (this *compressionWriter) WriteHeader(data []byte) (n int, err error) {
	return this.writer.Write(data)
}

// WriteHeaderLength 写入Header长度数据
func (this *compressionWriter) WriteHeaderLength(headerLength int) error {
	return nil
}

// WriteBodyLength 写入Body长度数据
func (this *compressionWriter) WriteBodyLength(bodyLength int64) error {
	return nil
}

func (this *compressionWriter) Write(data []byte) (n int, err error) {
	return this.writer.Write(data)
}

func (this *compressionWriter) Close() error {
	err := this.writer.Close()
	if err != nil {
		return err
	}
	return this.rawWriter.Close()
}

func (this *compressionWriter) Discard() error {
	err := this.writer.Close()
	if err != nil {
		return err
	}
	return this.rawWriter.Discard()
}

func (this *compressionWriter) Key() string {
	return this.key
}

func (this *compressionWriter) ExpiredAt() int64 {
	return this.expiredAt
}

func (this *compressionWriter) HeaderSize() int64 {
	return this.rawWriter.HeaderSize()
}

func (this *compressionWriter) BodySize() int64 {
	return this.rawWriter.BodySize()
}

// ItemType 内容类型
func (this *compressionWriter) ItemType() ItemType {
	return this.rawWriter.ItemType()
}
