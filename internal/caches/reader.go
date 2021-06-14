package caches

type ReaderFunc func(n int) (goNext bool, err error)

type Reader interface {
	// Init 初始化
	Init() error

	// TypeName 类型名称
	TypeName() string

	// Status 状态码
	Status() int

	// LastModified 最后修改时间
	LastModified() int64

	// ReadHeader 读取Header
	ReadHeader(buf []byte, callback ReaderFunc) error

	// ReadBody 读取Body
	ReadBody(buf []byte, callback ReaderFunc) error

	// ReadBodyRange 读取某个范围内的Body
	ReadBodyRange(buf []byte, start int64, end int64, callback ReaderFunc) error

	// HeaderSize Header Size
	HeaderSize() int64

	// BodySize Body Size
	BodySize() int64

	// Close 关闭
	Close() error
}
