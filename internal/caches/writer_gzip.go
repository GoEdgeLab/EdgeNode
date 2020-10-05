package caches

import "compress/gzip"

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
