// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package compressions

import (
	"github.com/klauspost/compress/zstd"
	"io"
)

type ZSTDReader struct {
	BaseReader

	reader *zstd.Decoder
}

func NewZSTDReader(reader io.Reader) (Reader, error) {
	return sharedZSTDReaderPool.Get(reader)
}

func newZSTDReader(reader io.Reader) (Reader, error) {
	r, err := zstd.NewReader(reader)
	if err != nil {
		return nil, err
	}
	return &ZSTDReader{
		reader: r,
	}, nil
}

func (this *ZSTDReader) Read(p []byte) (n int, err error) {
	return this.reader.Read(p)
}

func (this *ZSTDReader) Reset(reader io.Reader) error {
	return this.reader.Reset(reader)
}

func (this *ZSTDReader) RawClose() error {
	this.reader.Close()
	return nil
}

func (this *ZSTDReader) Close() error {
	return this.Finish(this)
}
