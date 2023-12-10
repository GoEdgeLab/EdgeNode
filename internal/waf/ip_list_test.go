// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package waf_test

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/firewallconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/waf"
	"github.com/iwind/TeaGo/Tea"
	"github.com/iwind/TeaGo/assert"
	_ "github.com/iwind/TeaGo/bootstrap"
	"github.com/iwind/TeaGo/logs"
	timeutil "github.com/iwind/TeaGo/utils/time"
	"os"
	"runtime"
	"strconv"
	"testing"
	"time"
)

func TestNewIPList(t *testing.T) {
	var list = waf.NewIPList(waf.IPListTypeDeny)
	list.Add(waf.IPTypeAll, firewallconfigs.FirewallScopeGlobal, 1, "127.0.0.1", time.Now().Unix())
	list.Add(waf.IPTypeAll, firewallconfigs.FirewallScopeGlobal, 1, "127.0.0.2", time.Now().Unix()+1)
	list.Add(waf.IPTypeAll, firewallconfigs.FirewallScopeGlobal, 1, "127.0.0.1", time.Now().Unix()+2)
	list.Add(waf.IPTypeAll, firewallconfigs.FirewallScopeService, 1, "127.0.0.3", time.Now().Unix()+3)
	list.Add(waf.IPTypeAll, firewallconfigs.FirewallScopeGlobal, 1, "127.0.0.10", time.Now().Unix()+10)

	list.RemoveIP("127.0.0.1", 1, false)

	logs.PrintAsJSON(list.IPMap(), t)
	logs.PrintAsJSON(list.IdMap(), t)
}

func TestIPList_Expire(t *testing.T) {
	var list = waf.NewIPList(waf.IPListTypeDeny)
	list.Add(waf.IPTypeAll, firewallconfigs.FirewallScopeGlobal, 1, "127.0.0.1", time.Now().Unix())
	list.Add(waf.IPTypeAll, firewallconfigs.FirewallScopeGlobal, 1, "127.0.0.2", time.Now().Unix()+1)
	list.Add(waf.IPTypeAll, firewallconfigs.FirewallScopeGlobal, 1, "127.0.0.1", time.Now().Unix()+2)
	list.Add(waf.IPTypeAll, firewallconfigs.FirewallScopeGlobal, 1, "127.0.0.3", time.Now().Unix()+3)
	list.Add(waf.IPTypeAll, firewallconfigs.FirewallScopeGlobal, 1, "127.0.0.10", time.Now().Unix()+6)

	var ticker = time.NewTicker(1 * time.Second)
	for range ticker.C {
		t.Log("====")
		logs.PrintAsJSON(list.IPMap(), t)
		logs.PrintAsJSON(list.IdMap(), t)
		if len(list.IdMap()) == 0 {
			break
		}
	}
}

func TestIPList_Contains(t *testing.T) {
	var a = assert.NewAssertion(t)

	var list = waf.NewIPList(waf.IPListTypeDeny)

	for i := 0; i < 1_0000; i++ {
		list.Add(waf.IPTypeAll, firewallconfigs.FirewallScopeGlobal, 1, "192.168.1."+strconv.Itoa(i), time.Now().Unix()+3600)
	}
	//list.RemoveIP("192.168.1.100")
	{
		a.IsTrue(list.Contains(waf.IPTypeAll, firewallconfigs.FirewallScopeGlobal, 1, "192.168.1.100"))
	}
	{
		a.IsFalse(list.Contains(waf.IPTypeAll, firewallconfigs.FirewallScopeGlobal, 1, "192.168.2.100"))
	}
}

func TestIPList_ContainsExpires(t *testing.T) {
	var list = waf.NewIPList(waf.IPListTypeDeny)

	for i := 0; i < 1_0000; i++ {
		list.Add(waf.IPTypeAll, firewallconfigs.FirewallScopeGlobal, 1, "192.168.1."+strconv.Itoa(i), time.Now().Unix()+3600)
	}
	// list.RemoveIP("192.168.1.100", 1, false)
	for _, ip := range []string{"192.168.1.100", "192.168.2.100"} {
		expiresAt, ok := list.ContainsExpires(waf.IPTypeAll, firewallconfigs.FirewallScopeGlobal, 1, ip)
		t.Log(ok, expiresAt, timeutil.FormatTime("Y-m-d H:i:s", expiresAt))
	}
}

func TestIPList_Save(t *testing.T) {
	var a = assert.NewAssertion(t)

	var list = waf.NewIPList(waf.IPListTypeAllow)
	list.Add(waf.IPTypeAll, firewallconfigs.FirewallScopeGlobal, 1, "192.168.1.100", time.Now().Unix()+3600)
	list.Add(waf.IPTypeAll, firewallconfigs.FirewallScopeGlobal, 2, "192.168.1.101", time.Now().Unix()+3600)

	var file = Tea.Root + "/data/waf.iplist.json"
	err := list.Save(file)
	if err != nil {
		t.Fatal(err)
	}

	var newList = waf.NewIPList(waf.IPListTypeAllow)
	err = newList.Load(file)
	if err != nil {
		t.Fatal(err)
	}

	a.IsTrue(newList.Contains(waf.IPTypeAll, firewallconfigs.FirewallScopeGlobal, 1, "192.168.1.100"))

	_ = os.Remove(file)
}

func BenchmarkIPList_Add(b *testing.B) {
	runtime.GOMAXPROCS(1)

	var list = waf.NewIPList(waf.IPListTypeDeny)
	for i := 0; i < b.N; i++ {
		list.Add(waf.IPTypeAll, firewallconfigs.FirewallScopeGlobal, 1, "192.168.1."+strconv.Itoa(i), time.Now().Unix()+3600)
	}
	b.Log(len(list.IPMap()))
}

func BenchmarkIPList_Has(b *testing.B) {
	runtime.GOMAXPROCS(1)

	var list = waf.NewIPList(waf.IPListTypeDeny)
	b.ResetTimer()

	for i := 0; i < 1_0000; i++ {
		list.Add(waf.IPTypeAll, firewallconfigs.FirewallScopeGlobal, 1, "192.168.1."+strconv.Itoa(i), time.Now().Unix()+3600)
	}

	for i := 0; i < b.N; i++ {
		list.Contains(waf.IPTypeAll, firewallconfigs.FirewallScopeGlobal, 1, "192.168.1.100")
	}
}
