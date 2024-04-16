// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package compressions

import (
	"github.com/klauspost/compress/zstd"
	"io"
)

type ZSTDWriter struct {
	BaseWriter

	writer *zstd.Encoder
	level  int
}

func NewZSTDWriter(writer io.Writer, level int) (Writer, error) {
	return sharedZSTDWriterPool.Get(writer, level)
}

func newZSTDWriter(writer io.Writer) (Writer, error) {
	var level = 1
	var zstdLevel = zstd.SpeedFastest

	zstdWriter, err := zstd.NewWriter(writer, zstd.WithEncoderLevel(zstdLevel), zstd.WithWindowSize(16<<10), zstd.WithLowerEncoderMem(true))
	if err != nil {
		return nil, err
	}

	return &ZSTDWriter{
		writer:    zstdWriter,
		level:     level,
	}, nil
}

func (this *ZSTDWriter) Write(p []byte) (int, error) {
	return this.writer.Write(p)
}

func (this *ZSTDWriter) Flush() error {
	return this.writer.Flush()
}

func (this *ZSTDWriter) Reset(writer io.Writer) {
	this.writer.Reset(writer)
}

func (this *ZSTDWriter) RawClose() error {
	return this.writer.Close()
}

func (this *ZSTDWriter) Close() error {
	return this.Finish(this)
}

func (this *ZSTDWriter) Level() int {
	return this.level
}
