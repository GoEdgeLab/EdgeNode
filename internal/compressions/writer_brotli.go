// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package compressions

import (
	"github.com/andybalholm/brotli"
	"io"
)

type BrotliWriter struct {
	writer *brotli.Writer
	level  int
}

func NewBrotliWriter(writer io.Writer, level int) (Writer, error) {
	if level <= 0 {
		level = brotli.BestSpeed
	} else if level > brotli.BestCompression {
		level = brotli.BestCompression
	}
	return &BrotliWriter{
		writer: brotli.NewWriterLevel(writer, level),
		level:  level,
	}, nil
}

func (this *BrotliWriter) Write(p []byte) (int, error) {
	return this.writer.Write(p)
}

func (this *BrotliWriter) Flush() error {
	return this.writer.Flush()
}

func (this *BrotliWriter) Close() error {
	return this.writer.Close()
}

func (this *BrotliWriter) Level() int {
	return this.level
}
