// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package firewalls

// FirewallInterface 防火墙接口
type FirewallInterface interface {
	// Name 名称
	Name() string

	// IsReady 是否已准备被调用
	IsReady() bool

	// IsMock 是否为模拟
	IsMock() bool

	// AllowPort 允许端口
	AllowPort(port int, protocol string) error

	// RemovePort 删除端口
	RemovePort(port int, protocol string) error

	// RejectSourceIP 拒绝某个源IP连接
	RejectSourceIP(ip string, timeoutSeconds int) error

	// DropSourceIP 丢弃某个源IP数据
	// ip 要封禁的IP
	// timeoutSeconds 过期时间
	// async 是否异步
	DropSourceIP(ip string, timeoutSeconds int, async bool) error

	// RemoveSourceIP 删除某个源IP
	RemoveSourceIP(ip string) error
}
