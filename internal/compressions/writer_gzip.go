// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package compressions

import (
	"compress/gzip"
	"io"
)

type GzipWriter struct {
	writer *gzip.Writer
	level  int
}

func NewGzipWriter(writer io.Writer, level int) (Writer, error) {
	if level <= 0 {
		level = gzip.BestSpeed
	} else if level > gzip.BestCompression {
		level = gzip.BestCompression
	}

	gzipWriter, err := gzip.NewWriterLevel(writer, level)
	if err != nil {
		return nil, err
	}

	return &GzipWriter{
		writer: gzipWriter,
		level:  level,
	}, nil
}

func (this *GzipWriter) Write(p []byte) (int, error) {
	return this.writer.Write(p)
}

func (this *GzipWriter) Flush() error {
	return this.writer.Flush()
}

func (this *GzipWriter) Close() error {
	return this.writer.Close()
}

func (this *GzipWriter) Level() int {
	return this.level
}
