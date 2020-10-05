package caches

// 缓存内容写入接口
type Writer interface {
	// 写入数据
	Write(data []byte) (n int, err error)

	// 关闭
	Close() error

	// 丢弃
	Discard() error

	// Key
	Key() string

	// 过期时间
	ExpiredAt() int64
}
