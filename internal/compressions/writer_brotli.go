// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package compressions

import (
	"github.com/andybalholm/brotli"
	"io"
)

type BrotliWriter struct {
	BaseWriter

	writer *brotli.Writer
	level  int
}

func NewBrotliWriter(writer io.Writer, level int) (Writer, error) {
	return sharedBrotliWriterPool.Get(writer, level)
}

func newBrotliWriter(writer io.Writer, level int) (*BrotliWriter, error) {
	if level <= 0 {
		level = brotli.BestSpeed
	} else if level > brotli.BestCompression {
		level = brotli.BestCompression
	}
	return &BrotliWriter{
		writer: brotli.NewWriterOptions(writer, brotli.WriterOptions{
			Quality: level,
			LGWin:   13, // TODO 在全局设置里可以设置此值
		}),
		level: level,
	}, nil
}

func (this *BrotliWriter) Write(p []byte) (int, error) {
	return this.writer.Write(p)
}

func (this *BrotliWriter) Flush() error {
	return this.writer.Flush()
}

func (this *BrotliWriter) Reset(newWriter io.Writer) {
	this.writer.Reset(newWriter)
}

func (this *BrotliWriter) RawClose() error {
	return this.writer.Close()
}

func (this *BrotliWriter) Close() error {
	return this.Finish(this)
}

func (this *BrotliWriter) Level() int {
	return this.level
}
