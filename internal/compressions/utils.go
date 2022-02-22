// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package compressions

import (
	"errors"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"io"
)

type ContentEncoding = string

const (
	ContentEncodingBr      ContentEncoding = "br"
	ContentEncodingGzip    ContentEncoding = "gzip"
	ContentEncodingDeflate ContentEncoding = "deflate"
)

var ErrNotSupportedContentEncoding = errors.New("not supported content encoding")

// AllEncodings 当前支持的所有编码
func AllEncodings() []ContentEncoding {
	return []ContentEncoding{ContentEncodingBr, ContentEncodingGzip, ContentEncodingDeflate}
}

// NewReader 获取Reader
func NewReader(reader io.Reader, contentEncoding ContentEncoding) (Reader, error) {
	switch contentEncoding {
	case ContentEncodingBr:
		return NewBrotliReader(reader)
	case ContentEncodingGzip:
		return NewGzipReader(reader)
	case ContentEncodingDeflate:
		return NewDeflateReader(reader)
	}
	return nil, ErrNotSupportedContentEncoding
}

// NewWriter 获取Writer
// TODO 考虑重用Writer
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
