// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package compressions

import (
	"errors"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"io"
)

func NewWriter(writer io.Writer, compressType serverconfigs.HTTPCompressionType, level int) (Writer, error) {
	switch compressType {
	case serverconfigs.HTTPCompressionTypeGzip:
		return NewGzipWriter(writer, level)
	case serverconfigs.HTTPCompressionTypeDeflate:
		return NewDeflateWriter(writer, level)
	case serverconfigs.HTTPCompressionTypeBrotli:
		return NewBrotliWriter(writer, level)
	}
	return nil, errors.New("invalid compression type '" + compressType + "'")
}
