package waf

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/iwind/TeaGo/assert"
	"net/http"
	"testing"
)

func TestWAF_MatchRequest(t *testing.T) {
	a := assert.NewAssertion(t)

	set := NewRuleSet()
	set.Name = "Name_Age"
	set.Connector = RuleConnectorAnd
	set.Rules = []*Rule{
		{
			Param:    "${arg.name}",
			Operator: RuleOperatorEqString,
			Value:    "lu",
		},
		{
			Param:    "${arg.age}",
			Operator: RuleOperatorEq,
			Value:    "20",
		},
	}
	set.AddAction(ActionBlock, nil)

	group := NewRuleGroup()
	group.AddRuleSet(set)
	group.IsInbound = true

	waf := NewWAF()
	waf.AddRuleGroup(group)
	errs := waf.Init()
	if len(errs) > 0 {
		t.Fatal(errs[0])
	}

	req, err := http.NewRequest(http.MethodGet, "http://teaos.cn/hello?name=lu&age=20", nil)
	if err != nil {
		t.Fatal(err)
	}
	goNext, _, _, set, err := waf.MatchRequest(requests.NewTestRequest(req), nil)
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
