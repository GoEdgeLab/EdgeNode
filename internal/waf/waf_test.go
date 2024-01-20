package waf_test

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/firewallconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/waf"
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/iwind/TeaGo/assert"
	"github.com/iwind/TeaGo/maps"
	"net/http"
	"testing"
)

func TestWAF_MatchRequest(t *testing.T) {
	var a = assert.NewAssertion(t)

	var set = waf.NewRuleSet()
	set.Name = "Name_Age"
	set.Connector = waf.RuleConnectorAnd
	set.Rules = []*waf.Rule{
		{
			Param:    "${arg.name}",
			Operator: waf.RuleOperatorEqString,
			Value:    "lu",
		},
		{
			Param:    "${arg.age}",
			Operator: waf.RuleOperatorEq,
			Value:    "20",
		},
	}
	set.AddAction(waf.ActionBlock, nil)

	var group = waf.NewRuleGroup()
	group.AddRuleSet(set)
	group.IsInbound = true

	var wafInstance = waf.NewWAF()
	wafInstance.AddRuleGroup(group)
	errs := wafInstance.Init()
	if len(errs) > 0 {
		t.Fatal(errs[0])
	}

	req, err := http.NewRequest(http.MethodGet, "http://teaos.cn/hello?name=lu&age=20", nil)
	if err != nil {
		t.Fatal(err)
	}
	result, err := wafInstance.MatchRequest(requests.NewTestRequest(req), nil, firewallconfigs.ServerCaptchaTypeNone)
	if err != nil {
		t.Fatal(err)
	}
	if set == nil {
		t.Log("not match")
		return
	}
	t.Log("goNext:", result.GoNext, "set:", set.Name)
	a.IsFalse(result.GoNext)
}

func TestWAF_MatchRequest_Allow(t *testing.T) {
	var a = assert.NewAssertion(t)

	var wafInstance = waf.NewWAF()

	{
		var set = waf.NewRuleSet()
		set.Id = 1
		set.Name = "set1"
		set.Connector = waf.RuleConnectorAnd
		set.Rules = []*waf.Rule{
			{
				Param:    "${requestPath}",
				Operator: waf.RuleOperatorMatch,
				Value:    "hello",
			},
		}
		set.AddAction(waf.ActionAllow, maps.Map{
			"scope": "global",
		})

		var group = waf.NewRuleGroup()
		group.Id = 1
		group.AddRuleSet(set)
		group.IsInbound = true

		wafInstance.AddRuleGroup(group)
	}

	{
		var set = waf.NewRuleSet()
		set.Id = 2
		set.Name = "set2"
		set.Connector = waf.RuleConnectorAnd
		set.Rules = []*waf.Rule{
			{
				Param:    "${requestPath}",
				Operator: waf.RuleOperatorMatch,
				Value:    "he",
			},
		}
		set.AddAction(waf.ActionAllow, maps.Map{
			"scope": "global",
		})

		var group = waf.NewRuleGroup()
		group.Id = 2
		group.AddRuleSet(set)
		group.IsInbound = true

		wafInstance.AddRuleGroup(group)
	}

	errs := wafInstance.Init()
	if len(errs) > 0 {
		t.Fatal(errs[0])
	}

	req, err := http.NewRequest(http.MethodGet, "http://teaos.cn/hello?name=lu&age=20", nil)
	if err != nil {
		t.Fatal(err)
	}
	result, err := wafInstance.MatchRequest(requests.NewTestRequest(req), nil, firewallconfigs.ServerCaptchaTypeNone)
	if err != nil {
		t.Fatal(err)
	}
	if result.Set == nil {
		t.Log("not match")
		return
	}
	t.Log("goNext:", result.GoNext, "set:", result.Set.Name)
	a.IsTrue(result.Set.Id == 1)
	a.IsTrue(result.GoNext)
	a.IsTrue(result.IsAllowed)
	a.IsTrue(result.AllowScope == "global")
}

func TestWAF_MatchRequest_Allow2(t *testing.T) {
	var a = assert.NewAssertion(t)

	var wafInstance = waf.NewWAF()

	{
		var set = waf.NewRuleSet()
		set.Id = 1
		set.Name = "set1"
		set.Connector = waf.RuleConnectorAnd
		set.Rules = []*waf.Rule{
			{
				Param:    "${requestPath}",
				Operator: waf.RuleOperatorMatch,
				Value:    "hello",
			},
		}
		set.AddAction(waf.ActionAllow, maps.Map{
			"scope": "group",
		})

		var group = waf.NewRuleGroup()
		group.Id = 1
		group.AddRuleSet(set)
		group.IsInbound = true

		wafInstance.AddRuleGroup(group)
	}

	{
		var set = waf.NewRuleSet()
		set.Id = 2
		set.Name = "set2"
		set.Connector = waf.RuleConnectorAnd
		set.Rules = []*waf.Rule{
			{
				Param:    "${requestPath}",
				Operator: waf.RuleOperatorMatch,
				Value:    "he",
			},
		}
		set.AddAction(waf.ActionAllow, maps.Map{
			"scope": "global",
		})

		var group = waf.NewRuleGroup()
		group.Id = 2
		group.AddRuleSet(set)
		group.IsInbound = true

		wafInstance.AddRuleGroup(group)
	}

	errs := wafInstance.Init()
	if len(errs) > 0 {
		t.Fatal(errs[0])
	}

	req, err := http.NewRequest(http.MethodGet, "http://teaos.cn/hello?name=lu&age=20", nil)
	if err != nil {
		t.Fatal(err)
	}
	result, err := wafInstance.MatchRequest(requests.NewTestRequest(req), nil, firewallconfigs.ServerCaptchaTypeNone)
	if err != nil {
		t.Fatal(err)
	}
	if result.Set == nil {
		t.Log("not match")
		return
	}
	t.Log("goNext:", result.GoNext, "set:", result.Set.Name)
	a.IsTrue(result.Set.Id == 2)
	a.IsTrue(result.GoNext)
	a.IsTrue(result.IsAllowed)
	a.IsTrue(result.AllowScope == "global")
}
