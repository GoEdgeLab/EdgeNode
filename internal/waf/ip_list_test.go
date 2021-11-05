// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package waf

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/firewallconfigs"
	"github.com/iwind/TeaGo/assert"
	"github.com/iwind/TeaGo/logs"
	"runtime"
	"strconv"
	"testing"
	"time"
)

func TestNewIPList(t *testing.T) {
	list := NewIPList()
	list.Add(IPTypeAll, firewallconfigs.FirewallScopeGlobal, 1, "127.0.0.1", time.Now().Unix())
	list.Add(IPTypeAll, firewallconfigs.FirewallScopeGlobal, 1, "127.0.0.2", time.Now().Unix()+1)
	list.Add(IPTypeAll, firewallconfigs.FirewallScopeGlobal, 1, "127.0.0.1", time.Now().Unix()+2)
	list.Add(IPTypeAll, firewallconfigs.FirewallScopeGlobal, 1, "127.0.0.3", time.Now().Unix()+3)
	list.Add(IPTypeAll, firewallconfigs.FirewallScopeGlobal, 1, "127.0.0.10", time.Now().Unix()+10)

	var ticker = time.NewTicker(1 * time.Second)
	for range ticker.C {
		t.Log("====")
		logs.PrintAsJSON(list.ipMap, t)
		logs.PrintAsJSON(list.idMap, t)
		if len(list.idMap) == 0 {
			break
		}
	}
}

func TestIPList_Contains(t *testing.T) {
	a := assert.NewAssertion(t)

	list := NewIPList()

	for i := 0; i < 1_0000; i++ {
		list.Add(IPTypeAll, firewallconfigs.FirewallScopeGlobal, 1, "192.168.1."+strconv.Itoa(i), time.Now().Unix()+3600)
	}
	//list.RemoveIP("192.168.1.100")
	a.IsTrue(list.Contains(IPTypeAll, firewallconfigs.FirewallScopeGlobal, 1, "192.168.1.100"))
	a.IsFalse(list.Contains(IPTypeAll, firewallconfigs.FirewallScopeGlobal, 1, "192.168.2.100"))
}

func BenchmarkIPList_Add(b *testing.B) {
	runtime.GOMAXPROCS(1)

	list := NewIPList()
	for i := 0; i < b.N; i++ {
		list.Add(IPTypeAll, firewallconfigs.FirewallScopeGlobal, 1, "192.168.1."+strconv.Itoa(i), time.Now().Unix()+3600)
	}
	b.Log(len(list.ipMap))
}

func BenchmarkIPList_Has(b *testing.B) {
	runtime.GOMAXPROCS(1)

	list := NewIPList()

	for i := 0; i < 1_0000; i++ {
		list.Add(IPTypeAll, firewallconfigs.FirewallScopeGlobal, 1, "192.168.1."+strconv.Itoa(i), time.Now().Unix()+3600)
	}

	for i := 0; i < b.N; i++ {
		list.Contains(IPTypeAll, firewallconfigs.FirewallScopeGlobal, 1, "192.168.1.100")
	}
}
