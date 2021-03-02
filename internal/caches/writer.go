package caches

// 缓存内容写入接口
type Writer interface {
	// 写入Header数据
	WriteHeader(data []byte) (n int, err error)

	// 写入Body数据
	Write(data []byte) (n int, err error)

	// 写入的Header数据大小
	HeaderSize() int64

	// 写入的Body数据大小
	BodySize() int64

	// 关闭
	Close() error

	// 丢弃
	Discard() error

	// Key
	Key() string

	// 过期时间
	ExpiredAt() int64

	// 内容类型
	ItemType() ItemType
}
