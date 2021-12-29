// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package compressions

import (
	"github.com/andybalholm/brotli"
	"io"
	"strings"
)

type BrotliReader struct {
	reader *brotli.Reader
}

func NewBrotliReader(reader io.Reader) (Reader, error) {
	return &BrotliReader{reader: brotli.NewReader(reader)}, nil
}

func (this *BrotliReader) Read(p []byte) (n int, err error) {
	n, err = this.reader.Read(p)
	if err != nil && strings.Contains(err.Error(), "excessive") {
		err = io.EOF
	}
	return
}

func (this *BrotliReader) Close() error {
	return nil
}
