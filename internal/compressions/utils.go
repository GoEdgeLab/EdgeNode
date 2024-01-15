// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package compressions

import (
	"errors"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"io"
	"net/http"
)

type ContentEncoding = string

const (
	ContentEncodingBr      ContentEncoding = "br"
	ContentEncodingGzip    ContentEncoding = "gzip"
	ContentEncodingDeflate ContentEncoding = "deflate"
	ContentEncodingZSTD    ContentEncoding = "zstd"
)

var ErrNotSupportedContentEncoding = errors.New("not supported content encoding")

// AllEncodings 当前支持的所有编码
func AllEncodings() []ContentEncoding {
	return []ContentEncoding{
		ContentEncodingBr,
		ContentEncodingGzip,
		ContentEncodingZSTD,
		ContentEncodingDeflate,
	}
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
	case ContentEncodingZSTD:
		return NewZSTDReader(reader)
	}
	return nil, ErrNotSupportedContentEncoding
}

// NewWriter 获取Writer
func NewWriter(writer io.Writer, compressType serverconfigs.HTTPCompressionType, level int) (Writer, error) {
	switch compressType {
	case serverconfigs.HTTPCompressionTypeGzip:
		return NewGzipWriter(writer, level)
	case serverconfigs.HTTPCompressionTypeDeflate:
		return NewDeflateWriter(writer, level)
	case serverconfigs.HTTPCompressionTypeBrotli:
		return NewBrotliWriter(writer, level)
	case serverconfigs.HTTPCompressionTypeZSTD:
		return NewZSTDWriter(writer, level)
	}
	return nil, errors.New("invalid compression type '" + compressType + "'")
}

// SupportEncoding 检查是否支持某个编码
func SupportEncoding(encoding string) bool {
	return encoding == ContentEncodingBr ||
		encoding == ContentEncodingGzip ||
		encoding == ContentEncodingDeflate ||
		encoding == ContentEncodingZSTD
}

// WrapHTTPResponse 包装http.Response对象
func WrapHTTPResponse(resp *http.Response) {
	if resp == nil {
		return
	}

	var contentEncoding = resp.Header.Get("Content-Encoding")
	if len(contentEncoding) == 0 || !SupportEncoding(contentEncoding) {
		return
	}

	reader, err := NewReader(resp.Body, contentEncoding)
	if err != nil {
		// unable to decode, we ignore the error
		return
	}
	resp.Header.Del("Content-Encoding")
	resp.Header.Del("Content-Length")
	resp.Body = reader
}
