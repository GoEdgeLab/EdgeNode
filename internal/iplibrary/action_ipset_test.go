package iplibrary

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/firewallconfigs"
	"github.com/iwind/TeaGo/maps"
	"testing"
	"time"
)

func TestIPSetAction_Init(t *testing.T) {
	action := NewIPSetAction()
	err := action.Init(&firewallconfigs.FirewallActionConfig{
		Params: maps.Map{
			"path":      "/usr/bin/iptables",
			"whiteName": "white-list",
			"blackName": "black-list",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ok")
}

func TestIPSetAction_AddItem(t *testing.T) {
	action := NewIPSetAction()
	action.config = &firewallconfigs.FirewallActionIPSetConfig{
		Path:      "/usr/bin/iptables",
		WhiteName: "white-list",
		BlackName: "black-list",
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

func TestIPSetAction_DeleteItem(t *testing.T) {
	action := NewIPSetAction()
	err := action.Init(&firewallconfigs.FirewallActionConfig{
		Params: maps.Map{
			"path":      "/usr/bin/firewalld",
			"whiteName": "white-list",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	err = action.DeleteItem(IPListTypeWhite, &pb.IPItem{
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
