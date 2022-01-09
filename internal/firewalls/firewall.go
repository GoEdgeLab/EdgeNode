// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package firewalls

import (
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
)

var currentFirewall FirewallInterface

// 初始化
func init() {
	events.On(events.EventLoaded, func() {
		var firewall = Firewall()
		if firewall.Name() == "mock" {
			remotelogs.Warn("FIREWALL", "'firewalld' on this system should be enabled to block attackers more effectively")
		} else {
			remotelogs.Println("FIREWALL", "found local firewall '"+firewall.Name()+"'")
		}
	})
}

// Firewall 查找当前系统中最适合的防火墙
func Firewall() FirewallInterface {
	if currentFirewall != nil {
		return currentFirewall
	}

	// firewalld
	{
		var firewalld = NewFirewalld()
		if firewalld.IsReady() {
			currentFirewall = firewalld
			return currentFirewall
		}
	}

	// 至少返回一个
	currentFirewall = NewMockFirewall()
	return currentFirewall
}
