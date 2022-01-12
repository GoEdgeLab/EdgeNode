package nodes

import "github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"

// ListenerInterface 各协议监听器的接口
type ListenerInterface interface {
	// Init 初始化
	Init()

	// Serve 监听
	Serve() error

	// Close 关闭
	Close() error

	// Reload 重载配置
	Reload(serverGroup *serverconfigs.ServerAddressGroup)

	// CountActiveConnections 获取当前活跃的连接数
	CountActiveConnections() int
}
