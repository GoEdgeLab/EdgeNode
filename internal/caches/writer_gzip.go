package caches

import (
	"compress/gzip"
)

type gzipWriter struct {
	rawWriter Writer
	writer    *gzip.Writer
	key       string
	expiredAt int64
}

func NewGzipWriter(gw Writer, key string, expiredAt int64) Writer {
	return &gzipWriter{
		rawWriter: gw,
		writer:    gzip.NewWriter(gw),
		key:       key,
		expiredAt: expiredAt,
	}
}

func (this *gzipWriter) WriteHeader(data []byte) (n int, err error) {
	return this.writer.Write(data)
}

// 写入Header长度数据
func (this *gzipWriter) WriteHeaderLength(headerLength int) error {
	return nil
}

// 写入Body长度数据
func (this *gzipWriter) WriteBodyLength(bodyLength int64) error {
	return nil
}

func (this *gzipWriter) Write(data []byte) (n int, err error) {
	return this.writer.Write(data)
}

func (this *gzipWriter) Close() error {
	err := this.writer.Close()
	if err != nil {
		return err
	}
	return this.rawWriter.Close()
}

func (this *gzipWriter) Discard() error {
	err := this.writer.Close()
	if err != nil {
		return err
	}
	return this.rawWriter.Discard()
}

func (this *gzipWriter) Key() string {
	return this.key
}

func (this *gzipWriter) ExpiredAt() int64 {
	return this.expiredAt
}

func (this *gzipWriter) HeaderSize() int64 {
	return this.rawWriter.HeaderSize()
}

func (this *gzipWriter) BodySize() int64 {
	return this.rawWriter.BodySize()
}

// 内容类型
func (this *gzipWriter) ItemType() ItemType {
	return this.rawWriter.ItemType()
}
