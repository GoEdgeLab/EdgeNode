package waf_test

import (
	"bytes"
	"github.com/TeaOSLab/EdgeNode/internal/waf"
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/cespare/xxhash"
	"github.com/iwind/TeaGo/assert"
	"github.com/iwind/TeaGo/maps"
	"net/http"
	"regexp"
	"runtime"
	"testing"
)

func TestRuleSet_MatchRequest(t *testing.T) {
	var set = waf.NewRuleSet()
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

	err := set.Init(nil)
	if err != nil {
		t.Fatal(err)
	}

	rawReq, err := http.NewRequest(http.MethodGet, "http://teaos.cn/hello?name=lu&age=20", nil)
	if err != nil {
		t.Fatal(err)
	}
	req := requests.NewTestRequest(rawReq)
	t.Log(set.MatchRequest(req))
}

func TestRuleSet_MatchRequest2(t *testing.T) {
	var a = assert.NewAssertion(t)

	var set = waf.NewRuleSet()
	set.Connector = waf.RuleConnectorOr

	set.Rules = []*waf.Rule{
		{
			Param:    "${arg.name}",
			Operator: waf.RuleOperatorEqString,
			Value:    "lu",
		},
		{
			Param:    "${arg.age}",
			Operator: waf.RuleOperatorEq,
			Value:    "21",
		},
	}

	err := set.Init(nil)
	if err != nil {
		t.Fatal(err)
	}

	rawReq, err := http.NewRequest(http.MethodGet, "http://teaos.cn/hello?name=lu&age=20", nil)
	if err != nil {
		t.Fatal(err)
	}
	req := requests.NewTestRequest(rawReq)
	a.IsTrue(set.MatchRequest(req))
}

func TestRuleSet_MatchRequest_Allow(t *testing.T) {
	var a = assert.NewAssertion(t)

	var set = waf.NewRuleSet()
	set.Connector = waf.RuleConnectorOr

	set.Rules = []*waf.Rule{
		{
			Param:    "${requestPath}",
			Operator: waf.RuleOperatorMatch,
			Value:    "hello",
		},
	}

	set.Actions = []*waf.ActionConfig{
		{
			Code: "allow",
			Options: maps.Map{
				"scope": waf.AllowScopeGroup,
			},
		},
	}

	var wafInstance = waf.NewWAF()

	err := set.Init(wafInstance)
	if err != nil {
		t.Fatal(err)
	}

	rawReq, err := http.NewRequest(http.MethodGet, "http://teaos.cn/hello?name=lu&age=20", nil)
	if err != nil {
		t.Fatal(err)
	}
	var req = requests.NewTestRequest(rawReq)
	b, _, err := set.MatchRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	a.IsTrue(b)

	var result = set.PerformActions(wafInstance, &waf.RuleGroup{}, req, nil)
	a.IsTrue(result.IsAllowed)
	t.Log("scope:", result.AllowScope)
}

func BenchmarkRuleSet_MatchRequest(b *testing.B) {
	runtime.GOMAXPROCS(1)

	var set = waf.NewRuleSet()
	set.Connector = waf.RuleConnectorOr

	set.Rules = []*waf.Rule{
		{
			Param:    "${requestAll}",
			Operator: waf.RuleOperatorMatch,
			Value:    `(onmouseover|onmousemove|onmousedown|onmouseup|onerror|onload|onclick|ondblclick|onkeydown|onkeyup|onkeypress)\s*=`,
		},
		{
			Param:    "${requestAll}",
			Operator: waf.RuleOperatorMatch,
			Value:    `\b(eval|system|exec|execute|passthru|shell_exec|phpinfo)\s*\(`,
		},
		{
			Param:    "${arg.name}",
			Operator: waf.RuleOperatorEqString,
			Value:    "lu",
		},
		{
			Param:    "${arg.age}",
			Operator: waf.RuleOperatorEq,
			Value:    "21",
		},
	}

	err := set.Init(nil)
	if err != nil {
		b.Fatal(err)
	}

	rawReq, err := http.NewRequest(http.MethodPost, "http://teaos.cn/hello?name=lu&age=20", bytes.NewBuffer(bytes.Repeat([]byte("HELLO"), 1024)))
	if err != nil {
		b.Fatal(err)
	}
	req := requests.NewTestRequest(rawReq)
	for i := 0; i < b.N; i++ {
		_, _, _ = set.MatchRequest(req)
	}
}

func BenchmarkRuleSet_MatchRequest_Regexp(b *testing.B) {
	runtime.GOMAXPROCS(1)

	var set = waf.NewRuleSet()
	set.Connector = waf.RuleConnectorOr

	set.Rules = []*waf.Rule{
		{
			Param:             "${requestBody}",
			Operator:          waf.RuleOperatorMatch,
			Value:             `\b(eval|system|exec|execute|passthru|shell_exec|phpinfo)\s*\(`,
			IsCaseInsensitive: false,
		},
	}

	err := set.Init(nil)
	if err != nil {
		b.Fatal(err)
	}

	rawReq, err := http.NewRequest(http.MethodPost, "http://teaos.cn/hello?name=lu&age=20", bytes.NewBuffer(bytes.Repeat([]byte("HELLO"), 2048)))
	if err != nil {
		b.Fatal(err)
	}
	req := requests.NewTestRequest(rawReq)
	for i := 0; i < b.N; i++ {
		_, _, _ = set.MatchRequest(req)
	}
}

func BenchmarkRuleSet_MatchRequest_Regexp2(b *testing.B) {
	reg, err := regexp.Compile(`(?iU)\b(eval|system|exec|execute|passthru|shell_exec|phpinfo)\b`)
	if err != nil {
		b.Fatal(err)
	}

	buf := bytes.Repeat([]byte(" HELLO "), 10240)

	for i := 0; i < b.N; i++ {
		_ = reg.Match(buf)
	}
}

func BenchmarkRuleSet_MatchRequest_Regexp3(b *testing.B) {
	reg, err := regexp.Compile(`(?iU)^(eval|system|exec|execute|passthru|shell_exec|phpinfo)`)
	if err != nil {
		b.Fatal(err)
	}

	buf := bytes.Repeat([]byte(" HELLO "), 1024)

	for i := 0; i < b.N; i++ {
		_ = reg.Match(buf)
	}
}

func BenchmarkHash(b *testing.B) {
	runtime.GOMAXPROCS(1)

	for i := 0; i < b.N; i++ {
		_ = xxhash.Sum64(bytes.Repeat([]byte("HELLO"), 10240))
	}
}
