package iplibrary

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/firewallconfigs"
	"testing"
	"time"
)

func TestIPTablesAction_AddItem(t *testing.T) {
	action := NewIPTablesAction()
	action.config = &firewallconfigs.FirewallActionIPTablesConfig{
		Path: "/usr/bin/iptables",
	}
	{
		err := action.AddItem(IPListTypeWhite, &pb.IPItem{
			Type:      "ipv4",
			Id:        1,
			IpFrom:    "192.168.1.100",
			ExpiredAt: time.Now().Unix() + 30,
		})
		if err != nil {
			t.Fatal(err)
		}
		t.Log("ok")
	}

	{
		err := action.AddItem(IPListTypeBlack, &pb.IPItem{
			Type:      "ipv4",
			Id:        1,
			IpFrom:    "192.168.1.100",
			ExpiredAt: time.Now().Unix() + 30,
		})
		if err != nil {
			t.Fatal(err)
		}
		t.Log("ok")
	}
}

func TestIPTablesAction_DeleteItem(t *testing.T) {
	action := NewIPTablesAction()
	action.config = &firewallconfigs.FirewallActionIPTablesConfig{
		Path: "/usr/bin/firewalld",
	}
	err := action.DeleteItem(IPListTypeWhite, &pb.IPItem{
		Type:      "ipv4",
		Id:        1,
		IpFrom:    "192.168.1.100",
		ExpiredAt: time.Now().Unix() + 30,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ok")
}
