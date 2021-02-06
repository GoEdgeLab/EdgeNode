package iplibrary

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/firewallconfigs"
	"testing"
	"time"
)

func TestFirewalldAction_AddItem(t *testing.T) {
	{
		action := NewFirewalldAction()
		action.config = &firewallconfigs.FirewallActionFirewalldConfig{
			Path: "/usr/bin/firewalld",
		}
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
		action := NewFirewalldAction()
		action.config = &firewallconfigs.FirewallActionFirewalldConfig{
			Path: "/usr/bin/firewalld",
		}
		err := action.AddItem(IPListTypeBlack, &pb.IPItem{
			Type:      "ipv4",
			Id:        1,
			IpFrom:    "192.168.1.101",
			ExpiredAt: time.Now().Unix() + 30,
		})
		if err != nil {
			t.Fatal(err)
		}
		t.Log("ok")
	}
}

func TestFirewalldAction_DeleteItem(t *testing.T) {
	action := NewFirewalldAction()
	action.config = &firewallconfigs.FirewallActionFirewalldConfig{
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

func TestFirewalldAction_MultipleItem(t *testing.T) {
	action := NewFirewalldAction()
	action.config = &firewallconfigs.FirewallActionFirewalldConfig{
		Path: "/usr/bin/firewalld",
	}
	err := action.AddItem(IPListTypeBlack, &pb.IPItem{
		Type:      "ipv4",
		Id:        1,
		IpFrom:    "192.168.1.30",
		IpTo:      "192.168.1.200",
		ExpiredAt: time.Now().Unix() + 30,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ok")
}
