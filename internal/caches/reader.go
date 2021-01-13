package caches

type ReaderFunc func(n int) (goNext bool, err error)

type Reader interface {
	// 初始化
	Init() error

	// 状态码
	Status() int

	// 读取Header
	ReadHeader(buf []byte, callback ReaderFunc) error

	// 读取Body
	ReadBody(buf []byte, callback ReaderFunc) error

	// 读取某个范围内的Body
	ReadBodyRange(buf []byte, start int64, end int64, callback ReaderFunc) error

	// Header Size
	HeaderSize() int64

	// Body Size
	BodySize() int64

	// 关闭
	Close() error
}
