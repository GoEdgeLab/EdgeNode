package iplibrary

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/firewallconfigs"
	"github.com/iwind/TeaGo/maps"
	"testing"
)

func TestActionManager_UpdateActions(t *testing.T) {
	manager := NewActionManager()
	manager.UpdateActions([]*firewallconfigs.FirewallActionConfig{
		{
			Id:   1,
			Type: "ipset",
			Params: maps.Map{
				"whiteName": "edge-white-list",
				"blackName": "edge-black-list",
			},
		},
	})
	t.Log("===config===")
	for _, c := range manager.configMap {
		t.Log(c.Id, c.Type)
	}
	t.Log("===instance===")
	for id, c := range manager.instanceMap {
		t.Log(id, c)
	}

	manager.UpdateActions([]*firewallconfigs.FirewallActionConfig{
		{
			Id:   1,
			Type: "ipset",
			Params: maps.Map{
				"whiteName": "edge-white-list",
				"blackName": "edge-black-list",
			},
		},
		{
			Id:   2,
			Type: "iptables",
			Params: maps.Map{
			},
		},
	})

	t.Log("===config===")
	for _, c := range manager.configMap {
		t.Log(c.Id, c.Type)
	}
	t.Log("===instance===")
	for id, c := range manager.instanceMap {
		t.Logf("%d: %#v", id, c)
	}

}
