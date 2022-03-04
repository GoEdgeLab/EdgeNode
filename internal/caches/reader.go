package caches

import "github.com/TeaOSLab/EdgeNode/internal/utils/ranges"

type ReaderFunc func(n int) (goNext bool, err error)

type Reader interface {
	// Init 初始化
	Init() error

	// TypeName 类型名称
	TypeName() string

	// ExpiresAt 过期时间
	ExpiresAt() int64

	// Status 状态码
	Status() int

	// LastModified 最后修改时间
	LastModified() int64

	// ReadHeader 读取Header
	ReadHeader(buf []byte, callback ReaderFunc) error

	// ReadBody 读取Body
	ReadBody(buf []byte, callback ReaderFunc) error

	// Read 实现io.Reader接口
	Read(buf []byte) (int, error)

	// ReadBodyRange 读取某个范围内的Body
	ReadBodyRange(buf []byte, start int64, end int64, callback ReaderFunc) error

	// HeaderSize Header Size
	HeaderSize() int64

	// BodySize Body Size
	BodySize() int64

	// ContainsRange 是否包含某个区间内容
	ContainsRange(r rangeutils.Range) (r2 rangeutils.Range, ok bool)

	// Close 关闭
	Close() error
}
