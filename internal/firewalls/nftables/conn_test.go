// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.
//go:build linux
// +build linux

package nftables_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/firewalls/nftables"
	executils "github.com/TeaOSLab/EdgeNode/internal/utils/exec"
	"testing"
)

func TestConn_Test(t *testing.T) {
	_, err := executils.LookPath("nft")
	if err == nil {
		t.Log("ok")
		return
	}
	t.Log(err)
}

func TestConn_GetTable_NotFound(t *testing.T) {
	conn, err := nftables.NewConn()
	if err != nil {
		t.Fatal(err)
	}

	table, err := conn.GetTable("a", nftables.TableFamilyIPv4)
	if err != nil {
		if err == nftables.ErrTableNotFound {
			t.Log("table not found")
		} else {
			t.Fatal(err)
		}
	} else {
		t.Log("table:", table)
	}
}

func TestConn_GetTable(t *testing.T) {
	conn, err := nftables.NewConn()
	if err != nil {
		t.Fatal(err)
	}

	table, err := conn.GetTable("myFilter", nftables.TableFamilyIPv4)
	if err != nil {
		if err == nftables.ErrTableNotFound {
			t.Log("table not found")
		} else {
			t.Fatal(err)
		}
	} else {
		t.Log("table:", table)
	}
}

func TestConn_AddTable(t *testing.T) {
	conn, err := nftables.NewConn()
	if err != nil {
		t.Fatal(err)
	}

	{
		table, err := conn.AddIPv4Table("test_ipv4")
		if err != nil {
			t.Fatal(err)
		}
		t.Log(table.Name())
	}
	{
		table, err := conn.AddIPv6Table("test_ipv6")
		if err != nil {
			t.Fatal(err)
		}
		t.Log(table.Name())
	}
}

func TestConn_DeleteTable(t *testing.T) {
	conn, err := nftables.NewConn()
	if err != nil {
		t.Fatal(err)
	}

	err = conn.DeleteTable("test_ipv4", nftables.TableFamilyIPv4)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ok")
}
