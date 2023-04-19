// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.
//go:build linux

package nftables_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/firewalls/nftables"
	"net"
	"testing"
)

func getIPv4Chain(t *testing.T) *nftables.Chain {
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

	chain, err := table.GetChain("test_chain")
	if err != nil {
		if err == nftables.ErrChainNotFound {
			chain, err = table.AddAcceptChain("test_chain")
		}
	}

	if err != nil {
		t.Fatal(err)
	}

	return chain
}

func TestChain_AddAcceptIPRule(t *testing.T) {
	var chain = getIPv4Chain(t)
	_, err := chain.AddAcceptIPv4Rule(net.ParseIP("192.168.2.40").To4(), nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestChain_AddDropIPRule(t *testing.T) {
	var chain = getIPv4Chain(t)
	_, err := chain.AddDropIPv4Rule(net.ParseIP("192.168.2.31").To4(), nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestChain_AddAcceptSetRule(t *testing.T) {
	var chain = getIPv4Chain(t)
	_, err := chain.AddAcceptIPv4SetRule("ipv4_black_set", nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestChain_AddDropSetRule(t *testing.T) {
	var chain = getIPv4Chain(t)
	_, err := chain.AddDropIPv4SetRule("ipv4_black_set", nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestChain_AddRejectSetRule(t *testing.T) {
	var chain = getIPv4Chain(t)
	_, err := chain.AddRejectIPv4SetRule("ipv4_black_set", nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestChain_GetRuleWithUserData(t *testing.T) {
	var chain = getIPv4Chain(t)
	rule, err := chain.GetRuleWithUserData([]byte("test"))
	if err != nil {
		if err == nftables.ErrRuleNotFound {
			t.Log("rule not found")
			return
		} else {
			t.Fatal(err)
		}
	}
	t.Log("rule:", rule)
}

func TestChain_GetRules(t *testing.T) {
	var chain = getIPv4Chain(t)
	rules, err := chain.GetRules()
	if err != nil {
		t.Fatal(err)
	}
	for _, rule := range rules {
		t.Log("handle:", rule.Handle(), "set name:", rule.LookupSetName(),
			"verdict:", rule.VerDict(), "user data:", string(rule.UserData()))
	}
}

func TestChain_DeleteRule(t *testing.T) {
	var chain = getIPv4Chain(t)
	rule, err := chain.GetRuleWithUserData([]byte("test"))
	if err != nil {
		if err == nftables.ErrRuleNotFound {
			t.Log("rule not found")
			return
		}
		t.Fatal(err)
	}
	err = chain.DeleteRule(rule)
	if err != nil {
		t.Fatal(err)
	}
}

func TestChain_Flush(t *testing.T) {
	var chain = getIPv4Chain(t)
	err := chain.Flush()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ok")
}
