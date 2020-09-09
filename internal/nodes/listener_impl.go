package nodes

// 各协议监听器的具体实现
type ListenerImpl interface {
	// 初始化
	Init()

	// 监听
	Serve() error

	// 关闭
	Close() error
}
