package waf

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/checkpoints"
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/iwind/TeaGo/assert"
	"github.com/iwind/TeaGo/maps"
	"net/http"
	"net/url"
	"testing"
)

func TestRule_Init_Single(t *testing.T) {
	rule := NewRule()
	rule.Param = "${arg.name}"
	rule.Operator = RuleOperatorEqString
	rule.Value = "lu"
	err := rule.Init()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(rule.singleParam, rule.singleCheckpoint)
	rawReq, err := http.NewRequest(http.MethodGet, "http://teaos.cn/hello?name=lu&age=20", nil)
	if err != nil {
		t.Fatal(err)
	}

	req := requests.NewTestRequest(rawReq)
	t.Log(rule.MatchRequest(req))
}

func TestRule_Init_Composite(t *testing.T) {
	rule := NewRule()
	rule.Param = "${arg.name} ${arg.age}"
	rule.Operator = RuleOperatorContains
	rule.Value = "lu"
	err := rule.Init()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(rule.singleParam, rule.singleCheckpoint)

	rawReq, err := http.NewRequest(http.MethodGet, "http://teaos.cn/hello?name=lu&age=20", nil)
	if err != nil {
		t.Fatal(err)
	}
	req := requests.NewTestRequest(rawReq)
	t.Log(rule.MatchRequest(req))
}

func TestRule_Test(t *testing.T) {
	var a = assert.NewAssertion(t)

	{
		var rule = NewRule()
		rule.Operator = RuleOperatorGt
		rule.Value = "123"
		err := rule.Init()
		if err != nil {
			t.Fatal(err)
		}
		a.IsTrue(rule.Test("124"))
		a.IsFalse(rule.Test("123"))
		a.IsFalse(rule.Test("122"))
		a.IsFalse(rule.Test("abcdef"))
	}

	{
		var rule = NewRule()
		rule.Operator = RuleOperatorGte
		rule.Value = "123"
		err := rule.Init()
		if err != nil {
			t.Fatal(err)
		}
		a.IsTrue(rule.Test("124"))
		a.IsTrue(rule.Test("123"))
		a.IsFalse(rule.Test("122"))
	}

	{
		var rule = NewRule()
		rule.Operator = RuleOperatorLt
		rule.Value = "123"
		err := rule.Init()
		if err != nil {
			t.Fatal(err)
		}
		a.IsFalse(rule.Test("124"))
		a.IsFalse(rule.Test("123"))
		a.IsTrue(rule.Test("122"))
	}

	{
		var rule = NewRule()
		rule.Operator = RuleOperatorLte
		rule.Value = "123"
		err := rule.Init()
		if err != nil {
			t.Fatal(err)
		}
		a.IsFalse(rule.Test("124"))
		a.IsTrue(rule.Test("123"))
		a.IsTrue(rule.Test("122"))
	}

	{
		var rule = NewRule()
		rule.Operator = RuleOperatorEq
		rule.Value = "123"
		err := rule.Init()
		if err != nil {
			t.Fatal(err)
		}
		a.IsFalse(rule.Test("124"))
		a.IsTrue(rule.Test("123"))
		a.IsFalse(rule.Test("122"))
	}

	{
		var rule = NewRule()
		rule.Operator = RuleOperatorNeq
		rule.Value = "123"
		err := rule.Init()
		if err != nil {
			t.Fatal(err)
		}
		a.IsTrue(rule.Test("124"))
		a.IsFalse(rule.Test("123"))
		a.IsTrue(rule.Test("122"))
	}

	{
		var rule = NewRule()
		rule.Operator = RuleOperatorEqString
		rule.Value = "123"
		err := rule.Init()
		if err != nil {
			t.Fatal(err)
		}
		a.IsFalse(rule.Test("124"))
		a.IsTrue(rule.Test("123"))
		a.IsFalse(rule.Test("122"))
	}

	{
		var rule = NewRule()
		rule.Operator = RuleOperatorEqString
		rule.Value = "abc"
		err := rule.Init()
		if err != nil {
			t.Fatal(err)
		}
		a.IsFalse(rule.Test("ABC"))
		a.IsTrue(rule.Test("abc"))
	}

	{
		var rule = NewRule()
		rule.Operator = RuleOperatorEqString
		rule.IsCaseInsensitive = true
		rule.Value = "abc"
		err := rule.Init()
		if err != nil {
			t.Fatal(err)
		}
		a.IsTrue(rule.Test("ABC"))
		a.IsTrue(rule.Test("abc"))
	}

	{
		var rule = NewRule()
		rule.Operator = RuleOperatorNeqString
		rule.Value = "abc"
		err := rule.Init()
		if err != nil {
			t.Fatal(err)
		}
		a.IsTrue(rule.Test("124"))
		a.IsFalse(rule.Test("abc"))
		a.IsTrue(rule.Test("122"))
	}

	{
		var rule = NewRule()
		rule.Operator = RuleOperatorNeqString
		rule.IsCaseInsensitive = true
		rule.Value = "abc"
		err := rule.Init()
		if err != nil {
			t.Fatal(err)
		}
		a.IsFalse(rule.Test("ABC"))
	}

	{
		var rule = NewRule()
		rule.Operator = RuleOperatorMatch
		rule.Value = "^\\d+"
		err := rule.Init()
		if err != nil {
			t.Fatal(err)
		}
		a.IsTrue(rule.Test("123"))
		a.IsFalse(rule.Test("abc123"))
	}

	{
		var rule = NewRule()
		rule.Operator = RuleOperatorMatch
		rule.Value = "^\\d+"
		err := rule.Init()
		if err != nil {
			t.Fatal(err)
		}
		a.IsTrue(rule.Test([]byte("123")))
		a.IsFalse(rule.Test([]byte("abc123")))
	}

	{
		var rule = NewRule()
		rule.Operator = RuleOperatorMatch
		rule.Value = "^\\d+"
		err := rule.Init()
		if err != nil {
			t.Fatal(err)
		}
		a.IsTrue(rule.Test([][]byte{[]byte("123"), []byte("456")}))
		a.IsFalse(rule.Test([][]byte{[]byte("abc123")}))
	}

	{
		var rule = NewRule()
		rule.Operator = RuleOperatorMatch
		rule.Value = "abc"
		rule.IsCaseInsensitive = true
		err := rule.Init()
		if err != nil {
			t.Fatal(err)
		}
		a.IsTrue(rule.Test("ABC"))
	}

	{
		var rule = NewRule()
		rule.Operator = RuleOperatorMatch
		rule.Value = "^\\d+"
		err := rule.Init()
		if err != nil {
			t.Fatal(err)
		}
		a.IsTrue(rule.Test([]string{"123", "456", "abc"}))
		a.IsFalse(rule.Test([]string{"abc123"}))
	}

	{
		var rule = NewRule()
		rule.Operator = RuleOperatorNotMatch
		rule.Value = "\\d+"
		err := rule.Init()
		if err != nil {
			t.Fatal(err)
		}
		a.IsFalse(rule.Test("123"))
		a.IsTrue(rule.Test("abc"))
	}

	{
		var rule = NewRule()
		rule.Operator = RuleOperatorNotMatch
		rule.Value = "abc"
		rule.IsCaseInsensitive = true
		err := rule.Init()
		if err != nil {
			t.Fatal(err)
		}
		a.IsFalse(rule.Test("ABC"))
	}

	{
		var rule = NewRule()
		rule.Operator = RuleOperatorNotMatch
		rule.Value = "^\\d+"
		err := rule.Init()
		if err != nil {
			t.Fatal(err)
		}
		a.IsFalse(rule.Test([]string{"123", "456", "abc"}))
		a.IsTrue(rule.Test([]string{"abc123"}))
	}

	{
		var rule = NewRule()
		rule.Operator = RuleOperatorNotMatch
		rule.Value = "^\\d+"
		err := rule.Init()
		if err != nil {
			t.Fatal(err)
		}
		a.IsFalse(rule.Test([][]byte{[]byte("123"), []byte("456")}))
		a.IsFalse(rule.Test([][]byte{[]byte("123"), []byte("abc")}))
		a.IsTrue(rule.Test([][]byte{[]byte("abc123")}))
	}

	{
		var rule = NewRule()
		rule.Operator = RuleOperatorMatch
		rule.Value = "^(?i)[a-z]+$"
		err := rule.Init()
		if err != nil {
			t.Fatal(err)
		}
		a.IsTrue(rule.Test("ABC"))
	}

	{
		var rule = NewRule()
		rule.Operator = RuleOperatorContains
		rule.Value = "Hello"
		err := rule.Init()
		if err != nil {
			t.Fatal(err)
		}
		a.IsTrue(rule.Test("Hello, World"))
	}

	{
		var rule = NewRule()
		rule.Operator = RuleOperatorContains
		rule.Value = "hello"
		rule.IsCaseInsensitive = true
		err := rule.Init()
		if err != nil {
			t.Fatal(err)
		}
		a.IsTrue(rule.Test("Hello, World"))
	}

	{
		var rule = NewRule()
		rule.Operator = RuleOperatorContains
		rule.Value = "Hello"
		err := rule.Init()
		if err != nil {
			t.Fatal(err)
		}
		a.IsTrue(rule.Test([]string{"Hello", "World"}))
		a.IsTrue(rule.Test(maps.Map{
			"a": "World", "b": "Hello",
		}))
		a.IsFalse(rule.Test(maps.Map{
			"a": "World", "b": "Hello2",
		}))
	}

	{
		var rule = NewRule()
		rule.Operator = RuleOperatorNotContains
		rule.Value = "Hello"
		err := rule.Init()
		if err != nil {
			t.Fatal(err)
		}
		a.IsFalse(rule.Test("Hello, World"))
		a.IsTrue(rule.Test("World"))
	}

	{
		var rule = NewRule()
		rule.Operator = RuleOperatorNotContains
		rule.Value = "hello"
		rule.IsCaseInsensitive = true
		err := rule.Init()
		if err != nil {
			t.Fatal(err)
		}
		a.IsFalse(rule.Test("Hello, World"))
		a.IsTrue(rule.Test("World"))
	}

	{
		var rule = NewRule()
		rule.Operator = RuleOperatorPrefix
		rule.Value = "Hello"
		err := rule.Init()
		if err != nil {
			t.Fatal(err)
		}
		a.IsTrue(rule.Test("Hello, World"))
		a.IsFalse(rule.Test("hello"))
		a.IsFalse(rule.Test("World, Hello"))
	}

	{
		var rule = NewRule()
		rule.Operator = RuleOperatorPrefix
		rule.Value = "hello"
		rule.IsCaseInsensitive = true
		err := rule.Init()
		if err != nil {
			t.Fatal(err)
		}
		a.IsTrue(rule.Test("Hello, World"))
		a.IsTrue(rule.Test("hello, World"))
		a.IsFalse(rule.Test("hell"))
		a.IsFalse(rule.Test("World, Hello"))
	}

	{
		var rule = NewRule()
		rule.Operator = RuleOperatorSuffix
		rule.Value = "Hello"
		err := rule.Init()
		if err != nil {
			t.Fatal(err)
		}
		a.IsFalse(rule.Test("Hello, World"))
		a.IsTrue(rule.Test("World, Hello"))
	}

	{
		var rule = NewRule()
		rule.Operator = RuleOperatorSuffix
		rule.Value = "hello"
		rule.IsCaseInsensitive = true
		err := rule.Init()
		if err != nil {
			t.Fatal(err)
		}
		a.IsFalse(rule.Test("Hello, World"))
		a.IsTrue(rule.Test("Hello"))
		a.IsFalse(rule.Test("llo"))
		a.IsTrue(rule.Test("World, Hello"))
	}

	{
		var rule = NewRule()
		rule.Operator = RuleOperatorHasKey
		rule.Value = "Hello"
		err := rule.Init()
		if err != nil {
			t.Fatal(err)
		}
		a.IsFalse(rule.Test("Hello, World"))
		a.IsTrue(rule.Test(maps.Map{
			"Hello": "World",
		}))
		a.IsFalse(rule.Test(maps.Map{
			"Hello1": "World",
		}))
	}

	{
		var rule = NewRule()
		rule.Operator = RuleOperatorHasKey
		rule.Value = "hello"
		rule.IsCaseInsensitive = true
		err := rule.Init()
		if err != nil {
			t.Fatal(err)
		}
		a.IsFalse(rule.Test("Hello, World"))
		a.IsTrue(rule.Test(maps.Map{
			"Hello": "World",
		}))
		a.IsFalse(rule.Test(maps.Map{
			"Hello1": "World",
		}))
	}

	{
		var rule = NewRule()
		rule.Operator = RuleOperatorHasKey
		rule.Value = "3"
		err := rule.Init()
		if err != nil {
			t.Fatal(err)
		}
		a.IsFalse(rule.Test("Hello, World"))
		a.IsFalse(rule.Test(maps.Map{
			"Hello": "World",
		}))
		a.IsTrue(rule.Test([]int{1, 2, 3, 4}))
	}
	{
		var rule = NewRule()
		rule.Operator = RuleOperatorContainsAnyWord
		rule.Value = "How\nare\nyou"
		rule.IsCaseInsensitive = true
		err := rule.Init()
		if err != nil {
			t.Fatal(err)
		}
		a.IsTrue(rule.Test("how"))
		a.IsTrue(rule.Test("How doing"))
		a.IsFalse(rule.Test("doing"))
	}
	{
		var rule = NewRule()
		rule.Operator = RuleOperatorContainsAllWords
		rule.Value = "How\nare\nyou"
		rule.IsCaseInsensitive = true
		err := rule.Init()
		if err != nil {
			t.Fatal(err)
		}
		a.IsTrue(rule.Test("how are you"))
		a.IsTrue(rule.Test("How are you doing"))
		a.IsFalse(rule.Test("How are dare"))
	}
	{
		var rule = NewRule()
		rule.Operator = RuleOperatorContainsSQLInjection
		err := rule.Init()
		if err != nil {
			t.Fatal(err)
		}
		a.IsTrue(rule.Test("id=123 OR 1=1"))
		a.IsTrue(rule.Test("id=456 UNION SELECT"))
		a.IsTrue(rule.Test("id=456 AND select load_file('') --"))
		a.IsFalse(rule.Test("id=123"))
		a.IsFalse(rule.Test("id=abc123 hello world '"))
	}
}

func TestRule_MatchStar(t *testing.T) {
	{
		rule := NewRule()
		rule.Operator = RuleOperatorMatch
		rule.Value = `/\*(!|\x00)`
		err := rule.Init()
		if err != nil {
			t.Fatal(err)
		}
		t.Log(rule.Test("/*!"))
		t.Log(rule.Test(url.QueryEscape("/*!")))
		t.Log(url.QueryEscape("/*!"))
	}
}

func TestRule_SetCheckpointFinder(t *testing.T) {
	{
		rule := NewRule()
		rule.Param = "${arg.abc}"
		rule.Operator = RuleOperatorMatch
		err := rule.Init()
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%#v", rule.singleCheckpoint)
	}

	{
		rule := NewRule()
		rule.Param = "${arg.abc}"
		rule.Operator = RuleOperatorMatch
		rule.checkpointFinder = func(prefix string) checkpoints.CheckpointInterface {
			return new(checkpoints.SampleRequestCheckpoint)
		}
		err := rule.Init()
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%#v", rule.singleCheckpoint)
	}
}

func TestRule_Version(t *testing.T) {
	a := assert.NewAssertion(t)

	{
		rule := Rule{
			Operator: RuleOperatorVersionRange,
			Value:    `1.0,1.1`,
		}
		a.IsNil(rule.Init())
		a.IsTrue(rule.Test("1.0"))
	}

	{
		rule := Rule{
			Operator: RuleOperatorVersionRange,
			Value:    `1.0,`,
		}
		a.IsNil(rule.Init())
		a.IsTrue(rule.Test("1.0"))
	}

	{
		rule := Rule{
			Operator: RuleOperatorVersionRange,
			Value:    `,1.1`,
		}
		a.IsNil(rule.Init())
		a.IsTrue(rule.Test("1.0"))
	}

	{
		rule := Rule{
			Operator: RuleOperatorVersionRange,
			Value:    `1.0,1.1`,
		}
		a.IsNil(rule.Init())
		a.IsFalse(rule.Test("0.9"))
	}

	{
		rule := Rule{
			Operator: RuleOperatorVersionRange,
			Value:    `1.0`,
		}
		a.IsNil(rule.Init())
		a.IsFalse(rule.Test("0.9"))
	}

	{
		rule := Rule{
			Operator: RuleOperatorVersionRange,
			Value:    `1.0`,
		}
		a.IsNil(rule.Init())
		a.IsTrue(rule.Test("1.1"))
	}
}

func TestRule_IP(t *testing.T) {
	a := assert.NewAssertion(t)

	{
		rule := Rule{
			Operator: RuleOperatorEqIP,
			Value:    "hello",
		}
		a.IsNotNil(rule.Init())
		a.IsFalse(rule.Test("hello"))
	}

	{
		rule := Rule{
			Operator: RuleOperatorEqIP,
			Value:    "hello",
		}
		a.IsNotNil(rule.Init())
		a.IsFalse(rule.Test("192.168.1.100"))
	}

	{
		rule := Rule{
			Operator: RuleOperatorEqIP,
			Value:    "192.168.1.100",
		}
		a.IsNil(rule.Init())
		a.IsTrue(rule.Test("192.168.1.100"))
	}

	{
		rule := Rule{
			Operator: RuleOperatorGtIP,
			Value:    "192.168.1.90",
		}
		a.IsNil(rule.Init())
		a.IsTrue(rule.Test("192.168.1.100"))
	}

	{
		rule := Rule{
			Operator: RuleOperatorGteIP,
			Value:    "192.168.1.90",
		}
		a.IsNil(rule.Init())
		a.IsTrue(rule.Test("192.168.1.100"))
	}

	{
		rule := Rule{
			Operator: RuleOperatorLtIP,
			Value:    "192.168.1.90",
		}
		a.IsNil(rule.Init())
		a.IsTrue(rule.Test("192.168.1.80"))
	}

	{
		rule := Rule{
			Operator: RuleOperatorLteIP,
			Value:    "192.168.1.90",
		}
		a.IsNil(rule.Init())
		a.IsTrue(rule.Test("192.168.0.100"))
	}

	{
		rule := Rule{
			Operator: RuleOperatorIPRange,
			Value:    "192.168.0.90,",
		}
		a.IsNil(rule.Init())
		a.IsFalse(rule.Test("192.168.0.100"))
	}

	{
		rule := Rule{
			Operator: RuleOperatorIPRange,
			Value:    "192.168.0.90,192.168.1.100",
		}
		a.IsNil(rule.Init())
		a.IsTrue(rule.Test("192.168.0.100"))
	}

	{
		var rule = Rule{
			Operator: RuleOperatorIPRange,
			Value:    ",192.168.1.100",
		}
		a.IsNil(rule.Init())
		a.IsTrue(rule.Test("192.168.0.100"))
	}

	{
		rule := Rule{
			Operator: RuleOperatorIPRange,
			Value:    "192.168.0.90,192.168.1.99",
		}
		a.IsNil(rule.Init())
		a.IsFalse(rule.Test("192.168.1.100"))
	}

	{
		rule := Rule{
			Operator: RuleOperatorIPRange,
			Value:    "192.168.0.90/24",
		}
		a.IsNil(rule.Init())
		a.IsFalse(rule.Test("192.168.1.100"))
	}

	{
		rule := Rule{
			Operator: RuleOperatorIPRange,
			Value:    "192.168.0.90/18",
		}
		a.IsNil(rule.Init())
		a.IsTrue(rule.Test("192.168.1.100"))
	}

	{
		rule := Rule{
			Operator: RuleOperatorIPRange,
			Value:    "a/18",
		}
		a.IsNil(rule.Init())
		a.IsFalse(rule.Test("192.168.1.100"))
	}

	{
		rule := Rule{
			Operator: RuleOperatorIPMod10,
			Value:    "6",
		}
		a.IsNil(rule.Init())
		a.IsTrue(rule.Test("192.168.1.100"))
	}

	{
		rule := Rule{
			Operator: RuleOperatorIPMod100,
			Value:    "76",
		}
		a.IsNil(rule.Init())
		a.IsTrue(rule.Test("192.168.1.100"))
	}

	{
		rule := Rule{
			Operator: RuleOperatorIPMod,
			Value:    "10,6",
		}
		a.IsNil(rule.Init())
		a.IsTrue(rule.Test("192.168.1.100"))
	}

	{
		rule := Rule{
			Operator: RuleOperatorNotIPRange,
			Value:    "192.168.0.90,192.168.1.100",
		}
		a.IsNil(rule.Init())
		a.IsFalse(rule.Test("192.168.0.100"))
	}
	{
		rule := Rule{
			Operator: RuleOperatorNotIPRange,
			Value:    "192.168.0.90,192.168.1.100",
		}
		a.IsNil(rule.Init())
		a.IsTrue(rule.Test("192.168.2.100"))
	}
	{
		rule := Rule{
			Operator: RuleOperatorNotIPRange,
			Value:    "192.168.0.90/8",
		}
		a.IsNil(rule.Init())
		a.IsFalse(rule.Test("192.168.2.100"))
	}
	{
		rule := Rule{
			Operator: RuleOperatorNotIPRange,
			Value:    "192.168.0.90/16",
		}
		a.IsNil(rule.Init())
		a.IsTrue(rule.Test("192.169.2.100"))
	}
}
