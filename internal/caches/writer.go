package caches

// Writer 缓存内容写入接口
type Writer interface {
	// WriteHeader 写入Header数据
	WriteHeader(data []byte) (n int, err error)

	// Write 写入Body数据
	Write(data []byte) (n int, err error)

	// WriteAt 在指定位置写入数据
	WriteAt(offset int64, data []byte) error

	// HeaderSize 写入的Header数据大小
	HeaderSize() int64

	// BodySize 写入的Body数据大小
	BodySize() int64

	// Close 关闭
	Close() error

	// Discard 丢弃
	Discard() error

	// Key Key
	Key() string

	// ExpiredAt 过期时间
	ExpiredAt() int64

	// ItemType 内容类型
	ItemType() ItemType
}
