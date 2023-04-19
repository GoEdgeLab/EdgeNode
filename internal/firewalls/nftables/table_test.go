// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.
//go:build linux

package nftables_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/firewalls/nftables"
	"testing"
)

func getIPv4Table(t *testing.T) *nftables.Table {
	conn, err := nftables.NewConn()
	if err != nil {
		t.Fatal(err)
	}
	table, err := conn.GetTable("test_ipv4", nftables.TableFamilyIPv4)
	if err != nil {
		if err == nftables.ErrTableNotFound {
			table, err = conn.AddIPv4Table("test_ipv4")
			if err != nil {
				t.Fatal(err)
			}
		} else {
			t.Fatal(err)
		}
	}
	return table
}

func TestTable_AddChain(t *testing.T) {
	var table = getIPv4Table(t)

	{
		chain, err := table.AddChain("test_default_chain", nil)
		if err != nil {
			t.Fatal(err)
		}
		t.Log("created:", chain.Name())
	}

	{
		chain, err := table.AddAcceptChain("test_accept_chain")
		if err != nil {
			t.Fatal(err)
		}
		t.Log("created:", chain.Name())
	}

	// Do not test drop chain before adding accept rule, you will drop yourself!!!!!!!
	/**{
		chain, err := table.AddDropChain("test_drop_chain")
		if err != nil {
			t.Fatal(err)
		}
		t.Log("created:", chain.Name())
	}**/
}

func TestTable_GetChain(t *testing.T) {
	var table = getIPv4Table(t)
	for _, chainName := range []string{"not_found_chain", "test_default_chain"} {
		chain, err := table.GetChain(chainName)
		if err != nil {
			if err == nftables.ErrChainNotFound {
				t.Log(chainName, ":", "not found")
			} else {
				t.Fatal(err)
			}
		} else {
			t.Log(chainName, ":", chain)
		}
	}
}

func TestTable_DeleteChain(t *testing.T) {
	var table = getIPv4Table(t)
	err := table.DeleteChain("test_default_chain")
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ok")
}

func TestTable_AddSet(t *testing.T) {
	var table = getIPv4Table(t)
	{
		set, err := table.AddSet("ipv4_black_set", &nftables.SetOptions{
			HasTimeout: false,
			KeyType:    nftables.TypeIPAddr,
		})
		if err != nil {
			t.Fatal(err)
		}
		t.Log(set.Name())
	}

	{
		set, err := table.AddSet("ipv6_black_set", &nftables.SetOptions{
			HasTimeout: true,
			//Timeout:    3600 * time.Second,
			KeyType: nftables.TypeIP6Addr,
		})
		if err != nil {
			t.Fatal(err)
		}
		t.Log(set.Name())
	}
}

func TestTable_GetSet(t *testing.T) {
	var table = getIPv4Table(t)
	for _, setName := range []string{"not_found_set", "ipv4_black_set"} {
		set, err := table.GetSet(setName)
		if err != nil {
			if err == nftables.ErrSetNotFound {
				t.Log(setName, ": not found")
			} else {
				t.Fatal(err)
			}
		} else {
			t.Log(setName, ":", set)
		}
	}
}

func TestTable_DeleteSet(t *testing.T) {
	var table = getIPv4Table(t)
	err := table.DeleteSet("ipv4_black_set")
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ok")
}

func TestTable_Flush(t *testing.T) {
	var table = getIPv4Table(t)
	err := table.Flush()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ok")
}
