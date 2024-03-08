// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package firewalls

import (
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"os"
	"runtime"
	"sync"
	"time"
)

var currentFirewall FirewallInterface
var firewallLocker = &sync.Mutex{}

// 初始化
func init() {
	if !teaconst.IsMain {
		return
	}

	events.On(events.EventLoaded, func() {
		var firewall = Firewall()
		if firewall.Name() != "mock" {
			remotelogs.Println("FIREWALL", "found local firewall '"+firewall.Name()+"'")
		}
	})
}

// Firewall 查找当前系统中最适合的防火墙
func Firewall() FirewallInterface {
	firewallLocker.Lock()
	defer firewallLocker.Unlock()
	if currentFirewall != nil {
		return currentFirewall
	}

	// http firewall
	{
		endpoint, _ := os.LookupEnv("EDGE_HTTP_FIREWALL_ENDPOINT")
		if len(endpoint) > 0 {
			var httpFirewall = NewHTTPFirewall(endpoint)
			for i := 0; i < 10; i++ {
				if httpFirewall.IsReady() {
					currentFirewall = httpFirewall
					remotelogs.Println("FIREWALL", "using http firewall '"+endpoint+"'")
					break
				}
				time.Sleep(1 * time.Second)
			}
			return httpFirewall
		}
	}

	// nftables
	if runtime.GOOS == "linux" {
		nftables, err := NewNFTablesFirewall()
		if err != nil {
			remotelogs.Warn("FIREWALL", "'nftables' should be installed on the system to enhance security (init failed: "+err.Error()+")")
		} else {
			if nftables.IsReady() {
				currentFirewall = nftables
				events.Notify(events.EventNFTablesReady)
				return nftables
			} else {
				remotelogs.Warn("FIREWALL", "'nftables' should be enabled on the system to enhance security")
			}
		}
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
