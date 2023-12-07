package waf

import (
	"bytes"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/firewallconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/iwind/TeaGo/assert"
	"github.com/iwind/TeaGo/lists"
	"github.com/iwind/TeaGo/logs"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func Test_Template(t *testing.T) {
	var a = assert.NewAssertion(t)

	var waf = Template()

	for _, group := range waf.Inbound {
		group.IsOn = true

		for _, set := range group.RuleSets {
			set.IsOn = true
		}
	}

	err := waf.Init()
	if err != nil {
		t.Fatal(err)
	}

	testTemplate1001(a, t, waf)
	testTemplate1002(a, t, waf)
	testTemplate1003(a, t, waf)
	testTemplate2001(a, t, waf)
	testTemplate3001(a, t, waf)
	testTemplate4001(a, t, waf)
	testTemplate5001(a, t, waf)
	testTemplate6001(a, t, waf)
	testTemplate7001(a, t, waf)
	testTemplate20001(a, t, waf)
}

func Test_Template2(t *testing.T) {
	reader := bytes.NewReader([]byte(strings.Repeat("HELLO", 1024)))
	req, err := http.NewRequest(http.MethodPost, "https://example.com/index.php?id=123", reader)
	if err != nil {
		t.Fatal(err)
	}

	waf := Template()
	var errs = waf.Init()
	if len(errs) > 0 {
		t.Fatal(errs[0])
	}

	now := time.Now()
	goNext, _, _, set, err := waf.MatchRequest(requests.NewTestRequest(req), nil, firewallconfigs.ServerCaptchaTypeNone)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(time.Since(now).Seconds()*1000, "ms")

	if goNext {
		t.Log("ok")
		return
	}

	logs.PrintAsJSON(set, t)
}

func BenchmarkTemplate(b *testing.B) {
	var waf = Template()

	for _, group := range waf.Inbound {
		group.IsOn = true

		for _, set := range group.RuleSets {
			set.IsOn = true
		}
	}

	err := waf.Init()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req, err := http.NewRequest(http.MethodGet, "https://example.com/index.php?id=123", nil)
		if err != nil {
			b.Fatal(err)
		}

		_, _, _, _, _ = waf.MatchRequest(requests.NewTestRequest(req), nil, firewallconfigs.ServerCaptchaTypeNone)
	}
}

func testTemplate1001(a *assert.Assertion, t *testing.T, template *WAF) {
	req, err := http.NewRequest(http.MethodGet, "http://example.com/index.php?id=onmousedown%3D123", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, result, err := template.MatchRequest(requests.NewTestRequest(req), nil, firewallconfigs.ServerCaptchaTypeNone)
	if err != nil {
		t.Fatal(err)
	}
	a.IsNotNil(result)
	if result != nil {
		a.IsTrue(result.Code == "1001")
	}
}

func testTemplate1002(a *assert.Assertion, t *testing.T, template *WAF) {
	req, err := http.NewRequest(http.MethodGet, "http://example.com/index.php?id=eval%28", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, result, err := template.MatchRequest(requests.NewTestRequest(req), nil, firewallconfigs.ServerCaptchaTypeNone)
	if err != nil {
		t.Fatal(err)
	}
	a.IsNotNil(result)
	if result != nil {
		a.IsTrue(result.Code == "1002")
	}
}

func testTemplate1003(a *assert.Assertion, t *testing.T, template *WAF) {
	req, err := http.NewRequest(http.MethodGet, "http://example.com/index.php?id=<script src=\"123.js\">", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, result, err := template.MatchRequest(requests.NewTestRequest(req), nil, firewallconfigs.ServerCaptchaTypeNone)
	if err != nil {
		t.Fatal(err)
	}
	a.IsNotNil(result)
	if result != nil {
		a.IsTrue(result.Code == "1003")
	}
}

func testTemplate2001(a *assert.Assertion, t *testing.T, template *WAF) {
	body := bytes.NewBuffer([]byte{})

	writer := multipart.NewWriter(body)

	{
		part, err := writer.CreateFormField("name")
		if err == nil {
			_, _ = part.Write([]byte("lu"))
		}
	}

	{
		part, err := writer.CreateFormField("age")
		if err == nil {
			_, _ = part.Write([]byte("20"))
		}
	}

	{
		part, err := writer.CreateFormFile("myFile", "hello.txt")
		if err == nil {
			_, _ = part.Write([]byte("Hello, World!"))
		}
	}

	{
		part, err := writer.CreateFormFile("myFile2", "hello.PHP")
		if err == nil {
			_, _ = part.Write([]byte("Hello, World, PHP!"))
		}
	}

	{
		part, err := writer.CreateFormFile("myFile3", "hello.asp")
		if err == nil {
			_, _ = part.Write([]byte("Hello, World, ASP Pages!"))
		}
	}

	{
		part, err := writer.CreateFormFile("myFile4", "hello.asp")
		if err == nil {
			_, _ = part.Write([]byte("Hello, World, ASP Pages!"))
		}
	}

	_ = writer.Close()

	req, err := http.NewRequest(http.MethodPost, "http://teaos.cn/", body)
	if err != nil {
		t.Fatal()
	}

	req.Header.Add("Content-Type", writer.FormDataContentType())

	_, _, _, result, err := template.MatchRequest(requests.NewTestRequest(req), nil, firewallconfigs.ServerCaptchaTypeNone)
	if err != nil {
		t.Fatal(err)
	}
	a.IsNotNil(result)
	if result != nil {
		a.IsTrue(result.Code == "2001")
	}
}

func testTemplate3001(a *assert.Assertion, t *testing.T, template *WAF) {
	req, err := http.NewRequest(http.MethodPost, "http://example.com/index.php?exec1+(", bytes.NewReader([]byte("exec('rm -rf /hello');")))
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, result, err := template.MatchRequest(requests.NewTestRequest(req), nil, firewallconfigs.ServerCaptchaTypeNone)
	if err != nil {
		t.Fatal(err)
	}
	a.IsNotNil(result)
	if result != nil {
		a.IsTrue(result.Code == "3001")
	}
}

func testTemplate4001(a *assert.Assertion, t *testing.T, template *WAF) {
	req, err := http.NewRequest(http.MethodPost, "http://example.com/index.php?whoami", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, result, err := template.MatchRequest(requests.NewTestRequest(req), nil, firewallconfigs.ServerCaptchaTypeNone)
	if err != nil {
		t.Fatal(err)
	}
	a.IsNotNil(result)
	if result != nil {
		a.IsTrue(result.Code == "4001")
	}
}

func testTemplate5001(a *assert.Assertion, t *testing.T, template *WAF) {
	{
		req, err := http.NewRequest(http.MethodPost, "http://example.com/.././..", nil)
		if err != nil {
			t.Fatal(err)
		}
		_, _, _, result, err := template.MatchRequest(requests.NewTestRequest(req), nil, firewallconfigs.ServerCaptchaTypeNone)
		if err != nil {
			t.Fatal(err)
		}
		a.IsNotNil(result)
		if result != nil {
			a.IsTrue(result.Code == "5001")
		}
	}

	{
		req, err := http.NewRequest(http.MethodPost, "http://example.com/..///./", nil)
		if err != nil {
			t.Fatal(err)
		}
		_, _, _, result, err := template.MatchRequest(requests.NewTestRequest(req), nil, firewallconfigs.ServerCaptchaTypeNone)
		if err != nil {
			t.Fatal(err)
		}
		a.IsNotNil(result)
		if result != nil {
			a.IsTrue(result.Code == "5001")
		}
	}
}

func testTemplate6001(a *assert.Assertion, t *testing.T, template *WAF) {
	{
		req, err := http.NewRequest(http.MethodPost, "http://example.com/.svn/123.txt", nil)
		if err != nil {
			t.Fatal(err)
		}
		_, _, _, result, err := template.MatchRequest(requests.NewTestRequest(req), nil, firewallconfigs.ServerCaptchaTypeNone)
		if err != nil {
			t.Fatal(err)
		}
		a.IsNotNil(result)
		if result != nil {
			a.IsTrue(result.Code == "6001")
		}
	}

	{
		req, err := http.NewRequest(http.MethodPost, "http://example.com/123.git", nil)
		if err != nil {
			t.Fatal(err)
		}
		_, _, _, result, err := template.MatchRequest(requests.NewTestRequest(req), nil, firewallconfigs.ServerCaptchaTypeNone)
		if err != nil {
			t.Fatal(err)
		}
		a.IsNil(result)
	}
}

func testTemplate7001(a *assert.Assertion, t *testing.T, template *WAF) {
	for _, id := range []string{
		"union select",
		" and if(",
		"/*!",
		" and select ",
		" and id=123 ",
		"(case when a=1 then ",
		"updatexml (",
		"; delete from table",
	} {
		req, err := http.NewRequest(http.MethodPost, "http://example.com/?id="+url.QueryEscape(id), nil)
		if err != nil {
			t.Fatal(err)
		}
		_, _, _, result, err := template.MatchRequest(requests.NewTestRequest(req), nil, firewallconfigs.ServerCaptchaTypeNone)
		if err != nil {
			t.Fatal(err)
		}
		a.IsNotNil(result)
		if result != nil {
			a.IsTrue(lists.ContainsAny([]string{"7001", "7002", "7003", "7004", "7005"}, result.Code))
		} else {
			t.Log("break:", id)
		}
	}
}

func TestTemplateSQLInjection(t *testing.T) {
	var template = Template()
	errs := template.Init()
	if len(errs) > 0 {
		t.Fatal(errs)
		return
	}
	var group = template.FindRuleGroupWithCode("sqlInjection")
	if group == nil {
		t.Fatal("group not found")
		return
	}
	//
	//for _, set := range group.RuleSets {
	//	for _, rule := range set.Rules {
	//		t.Logf("%#v", rule.singleCheckpoint)
	//	}
	//}

	req, err := http.NewRequest(http.MethodPost, "https://example.com/?id=1234", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, _, result, err := group.MatchRequest(requests.NewTestRequest(req))
	if err != nil {
		t.Fatal(err)
	}
	if result != nil {
		t.Log(result)
	}
}

func BenchmarkTemplateSQLInjection(b *testing.B) {
	var template = Template()
	errs := template.Init()
	if len(errs) > 0 {
		b.Fatal(errs)
		return
	}
	var group = template.FindRuleGroupWithCode("sqlInjection")
	if group == nil {
		b.Fatal("group not found")
		return
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, err := http.NewRequest(http.MethodPost, "https://example.com/?id=1234", nil)
			if err != nil {
				b.Fatal(err)
			}
			_, _, result, err := group.MatchRequest(requests.NewTestRequest(req))
			if err != nil {
				b.Fatal(err)
			}
			_ = result
		}
	})
}

func testTemplate20001(a *assert.Assertion, t *testing.T, template *WAF) {
	// enable bot rule set
	for _, g := range template.Inbound {
		if g.Code == "bot" {
			g.IsOn = true
			break
		}
	}

	for _, bot := range []string{
		"Googlebot",
		"AdsBot",
		"bingbot",
		"BingPreview",
		"facebookexternalhit",
		"Slurp",
		"Sogou",
		"Baiduspider http://baidu.com",
	} {
		req, err := http.NewRequest(http.MethodPost, "http://example.com/", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("User-Agent", bot)
		_, _, _, result, err := template.MatchRequest(requests.NewTestRequest(req), nil, firewallconfigs.ServerCaptchaTypeNone)
		if err != nil {
			t.Fatal(err)
		}
		a.IsNotNil(result)
		if result != nil {
			a.IsTrue(lists.ContainsAny([]string{"20001"}, result.Code))
		} else {
			t.Log("break:", bot)
		}
	}
}
