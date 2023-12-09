package waf_test

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/firewallconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/waf"
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/iwind/TeaGo/assert"
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
	goNext, _, _, set, err := wafInstance.MatchRequest(requests.NewTestRequest(req), nil, firewallconfigs.ServerCaptchaTypeNone)
	if err != nil {
		t.Fatal(err)
	}
	if set == nil {
		t.Log("not match")
		return
	}
	t.Log("goNext:", goNext, "set:", set.Name)
	a.IsFalse(goNext)
}
