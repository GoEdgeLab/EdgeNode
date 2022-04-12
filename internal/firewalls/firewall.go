// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.
//go:build !plus
// +build !plus

package firewalls

import (
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"runtime"
)

var currentFirewall FirewallInterface

// 初始化
func init() {
	events.On(events.EventLoaded, func() {
		var firewall = Firewall()
		if firewall.Name() != "mock" {
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
	if runtime.GOOS == "linux" {
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
