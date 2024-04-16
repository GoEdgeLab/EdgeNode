// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package compressions

import (
	"errors"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	memutils "github.com/TeaOSLab/EdgeNode/internal/utils/mem"
	"io"
	"net/http"
	"runtime"
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

// 系统CPU线程数
var countCPU = runtime.NumCPU()

// GenerateCompressLevel 根据系统资源自动生成压缩级别
func GenerateCompressLevel(minLevel int, maxLevel int) (level int) {
	if countCPU < 16 {
		return minLevel
	}

	if countCPU < 32 {
		return min(3, maxLevel)
	}

	return min(5, maxLevel)
}

// CalculatePoolSize 计算Pool尺寸
func CalculatePoolSize() int {
	var maxSize = memutils.SystemMemoryGB() * 64
	if maxSize == 0 {
		maxSize = 128
	}
	if maxSize > 4096 {
		maxSize = 4096
	}
	return maxSize
}
