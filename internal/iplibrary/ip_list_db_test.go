// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package iplibrary_test

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeNode/internal/iplibrary"
	_ "github.com/iwind/TeaGo/bootstrap"
	"github.com/iwind/TeaGo/logs"
	"testing"
	"time"
)

func TestIPListDB_AddItem(t *testing.T) {
	db, err := iplibrary.NewIPListDB()
	if err != nil {
		t.Fatal(err)
	}

	err = db.AddItem(&pb.IPItem{
		Id:                            1,
		IpFrom:                        "192.168.1.101",
		IpTo:                          "",
		Version:                       1024,
		ExpiredAt:                     time.Now().Unix() + 3600,
		Reason:                        "",
		ListId:                        2,
		IsDeleted:                     false,
		Type:                          "ipv4",
		EventLevel:                    "error",
		ListType:                      "black",
		IsGlobal:                      true,
		CreatedAt:                     0,
		NodeId:                        11,
		ServerId:                      22,
		SourceNodeId:                  0,
		SourceServerId:                0,
		SourceHTTPFirewallPolicyId:    0,
		SourceHTTPFirewallRuleGroupId: 0,
		SourceHTTPFirewallRuleSetId:   0,
		SourceServer:                  nil,
		SourceHTTPFirewallPolicy:      nil,
		SourceHTTPFirewallRuleGroup:   nil,
		SourceHTTPFirewallRuleSet:     nil,
	})
	if err != nil {
		t.Fatal(err)
	}

	err = db.Close()
	if err != nil {
		t.Fatal(err)
	}

	t.Log("ok")
}

func TestIPListDB_ReadItems(t *testing.T) {
	db, err := iplibrary.NewIPListDB()
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		_ = db.Close()
	}()

	items, err := db.ReadItems(0, 2)
	if err != nil {
		t.Fatal(err)
	}
	logs.PrintAsJSON(items, t)
}

func TestIPListDB_ReadMaxVersion(t *testing.T) {
	db, err := iplibrary.NewIPListDB()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(db.ReadMaxVersion())
}

func TestIPListDB_UpdateMaxVersion(t *testing.T) {
	db, err := iplibrary.NewIPListDB()
	if err != nil {
		t.Fatal(err)
	}
	err = db.UpdateMaxVersion(1027)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(db.ReadMaxVersion())
}
