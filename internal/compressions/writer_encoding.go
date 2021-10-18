// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package compressions

import (
	"bytes"
	"io"
)

type EncodingWriter struct {
	contentEncoding ContentEncoding
	writer          Writer
	buf             *bytes.Buffer
}

func NewEncodingWriter(contentEncoding ContentEncoding, writer Writer) (Writer, error) {
	return &EncodingWriter{
		contentEncoding: contentEncoding,
		writer:          writer,
		buf:             &bytes.Buffer{},
	}, nil
}

func (this *EncodingWriter) Write(p []byte) (int, error) {
	return this.buf.Write(p)
}

func (this *EncodingWriter) Flush() error {
	return this.writer.Flush()
}

func (this *EncodingWriter) Close() error {
	reader, err := NewReader(this.buf, this.contentEncoding)
	if err != nil {
		_ = this.writer.Close()
		return err
	}
	_, err = io.Copy(this.writer, reader)
	if err != nil {
		_ = reader.Close()
		_ = this.writer.Close()
		return err
	}

	_ = reader.Close()
	return this.writer.Close()
}

func (this *EncodingWriter) Level() int {
	return this.writer.Level()
}
