package nodes

import "github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"

// 各协议监听器的接口
type ListenerInterface interface {
	// 初始化
	Init()

	// 监听
	Serve() error

	// 关闭
	Close() error

	// 重载配置
	Reload(serverGroup *serverconfigs.ServerGroup)

	// 获取当前活跃的连接数
	CountActiveListeners() int
}
