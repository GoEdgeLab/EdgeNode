// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package compressions

import (
	"compress/flate"
	"io"
)

type DeflateWriter struct {
	writer *flate.Writer
	level  int
}

func NewDeflateWriter(writer io.Writer, level int) (Writer, error) {
	if level <= 0 {
		level = flate.BestSpeed
	} else if level > flate.BestCompression {
		level = flate.BestCompression
	}

	flateWriter, err := flate.NewWriter(writer, level)
	if err != nil {
		return nil, err
	}

	return &DeflateWriter{
		writer: flateWriter,
		level:  level,
	}, nil
}

func (this *DeflateWriter) Write(p []byte) (int, error) {
	return this.writer.Write(p)
}

func (this *DeflateWriter) Flush() error {
	return this.writer.Flush()
}

func (this *DeflateWriter) Close() error {
	return this.writer.Close()
}

func (this *DeflateWriter) Level() int {
	return this.level
}
