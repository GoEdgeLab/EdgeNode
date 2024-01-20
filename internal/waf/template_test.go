package waf_test

import (
	"bytes"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/firewallconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/waf"
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/iwind/TeaGo/assert"
	"github.com/iwind/TeaGo/lists"
	"github.com/iwind/TeaGo/logs"
	"github.com/iwind/TeaGo/types"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"testing"
	"time"
)

const testUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_0_0) AppleWebKit/500.00 (KHTML, like Gecko) Chrome/100.0.0.0"

func Test_Template(t *testing.T) {
	var a = assert.NewAssertion(t)

	wafInstance, err := waf.Template()
	if err != nil {
		t.Fatal(err)
	}

	testTemplate1010(a, t, wafInstance)
	testTemplate2001(a, t, wafInstance)
	testTemplate3001(a, t, wafInstance)
	testTemplate4001(a, t, wafInstance)
	testTemplate5001(a, t, wafInstance)
	testTemplate6001(a, t, wafInstance)
	testTemplate7010(a, t, wafInstance)
	testTemplate20001(a, t, wafInstance)
}

func Test_Template2(t *testing.T) {
	reader := bytes.NewReader([]byte(strings.Repeat("HELLO", 1024)))
	req, err := http.NewRequest(http.MethodPost, "https://example.com/index.php?id=123", reader)
	if err != nil {
		t.Fatal(err)
	}

	wafInstance, err := waf.Template()
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	result, err := wafInstance.MatchRequest(requests.NewTestRequest(req), nil, firewallconfigs.ServerCaptchaTypeNone)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(time.Since(now).Seconds()*1000, "ms")

	if result.GoNext {
		t.Log("ok")
		return
	}

	logs.PrintAsJSON(result.Set, t)
}

func BenchmarkTemplate(b *testing.B) {
	runtime.GOMAXPROCS(4)

	wafInstance, err := waf.Template()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, err := http.NewRequest(http.MethodGet, "https://example.com/index.php?id=123"+types.String(rand.Int()%10000), nil)
			if err != nil {
				b.Fatal(err)
			}
			req.Header.Set("User-Agent", testUserAgent)

			_, _ = wafInstance.MatchRequest(requests.NewTestRequest(req), nil, firewallconfigs.ServerCaptchaTypeNone)
		}
	})
}

func testTemplate1010(a *assert.Assertion, t *testing.T, template *waf.WAF) {
	for _, id := range []string{
		"<script",
		"<script src=\"123.js\">",
		"<script>alert(123)</script>",
		"<link",
		"<link>",
		"1 onfocus='alert(document.cookie)'",
	} {
		req, err := http.NewRequest(http.MethodGet, "https://example.com/index.php?id="+id, nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("User-Agent", testUserAgent)
		result, err := template.MatchRequest(requests.NewTestRequest(req), nil, firewallconfigs.ServerCaptchaTypeNone)
		if err != nil {
			t.Fatal(err)
		}
		a.IsNotNil(result.Set)
		if result.Set != nil {
			a.IsTrue(result.Set.Code == "1010")
		} else {
			t.Log("break at:", id)
		}
	}

	for _, id := range []string{
		"123",
		"abc",
		"<html></html>",
	} {
		req, err := http.NewRequest(http.MethodGet, "https://example.com/index.php?id="+url.QueryEscape(id), nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("User-Agent", testUserAgent)
		result, err := template.MatchRequest(requests.NewTestRequest(req), nil, firewallconfigs.ServerCaptchaTypeNone)
		if err != nil {
			t.Fatal(err)
		}
		a.IsNil(result.Set)
		if result.Set != nil {
			a.IsTrue(result.Set.Code == "1010")
		}
	}
}

func testTemplate2001(a *assert.Assertion, t *testing.T, template *waf.WAF) {
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

	result, err := template.MatchRequest(requests.NewTestRequest(req), nil, firewallconfigs.ServerCaptchaTypeNone)
	if err != nil {
		t.Fatal(err)
	}
	a.IsNotNil(result.Set)
	if result.Set != nil {
		a.IsTrue(result.Set.Code == "2001")
	}
}

func testTemplate3001(a *assert.Assertion, t *testing.T, template *waf.WAF) {
	req, err := http.NewRequest(http.MethodPost, "http://example.com/index.php?exec1+(", bytes.NewReader([]byte("exec('rm -rf /hello');")))
	if err != nil {
		t.Fatal(err)
	}
	result, err := template.MatchRequest(requests.NewTestRequest(req), nil, firewallconfigs.ServerCaptchaTypeNone)
	if err != nil {
		t.Fatal(err)
	}
	a.IsNotNil(result.Set)
	if result.Set != nil {
		a.IsTrue(result.Set.Code == "3001")
	}
}

func testTemplate4001(a *assert.Assertion, t *testing.T, template *waf.WAF) {
	req, err := http.NewRequest(http.MethodPost, "http://example.com/index.php?whoami", nil)
	if err != nil {
		t.Fatal(err)
	}
	result, err := template.MatchRequest(requests.NewTestRequest(req), nil, firewallconfigs.ServerCaptchaTypeNone)
	if err != nil {
		t.Fatal(err)
	}
	a.IsNotNil(result.Set)
	if result.Set != nil {
		a.IsTrue(result.Set.Code == "4001")
	}
}

func testTemplate5001(a *assert.Assertion, t *testing.T, template *waf.WAF) {
	{
		req, err := http.NewRequest(http.MethodPost, "http://example.com/.././..", nil)
		if err != nil {
			t.Fatal(err)
		}
		result, err := template.MatchRequest(requests.NewTestRequest(req), nil, firewallconfigs.ServerCaptchaTypeNone)
		if err != nil {
			t.Fatal(err)
		}
		a.IsNotNil(result.Set)
		if result.Set != nil {
			a.IsTrue(result.Set.Code == "5001")
		}
	}

	{
		req, err := http.NewRequest(http.MethodPost, "http://example.com/..///./", nil)
		if err != nil {
			t.Fatal(err)
		}
		result, err := template.MatchRequest(requests.NewTestRequest(req), nil, firewallconfigs.ServerCaptchaTypeNone)
		if err != nil {
			t.Fatal(err)
		}
		a.IsNotNil(result.Set)
		if result.Set != nil {
			a.IsTrue(result.Set.Code == "5001")
		}
	}
}

func testTemplate6001(a *assert.Assertion, t *testing.T, template *waf.WAF) {
	{
		req, err := http.NewRequest(http.MethodPost, "http://example.com/.svn/123.txt", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("User-Agent", testUserAgent)
		result, err := template.MatchRequest(requests.NewTestRequest(req), nil, firewallconfigs.ServerCaptchaTypeNone)
		if err != nil {
			t.Fatal(err)
		}
		a.IsNotNil(result.Set)
		if result.Set != nil {
			a.IsTrue(result.Set.Code == "6001")
		}
	}

	{
		req, err := http.NewRequest(http.MethodPost, "http://example.com/123.git", nil)
		if err != nil {
			t.Fatal(err)
		}
		result, err := template.MatchRequest(requests.NewTestRequest(req), nil, firewallconfigs.ServerCaptchaTypeNone)
		if err != nil {
			t.Fatal(err)
		}
		a.IsNotNil(result.Set)
	}
}

func testTemplate7010(a *assert.Assertion, t *testing.T, template *waf.WAF) {
	for _, id := range []string{
		" union all select id from credits",
		"' or 1=1",
		"' or '1'='1",
		"1' or '1'='1')) /*",
		"OR 1/** this is comment **/=1",
		"AND 1=2",
		"; INSERT INTO users (...)",
		"order by 10--",
		"UNION SELECT 1,null,null--",
		"' AND ASCII(SUBSTRING(username, 1, 1))=97 AND '1'='1",
		"||UTL_INADDR.GET_HOST_NAME((SELECT user FROM dual) )--",
		" AND IF(version() like '5%', sleep(10), 'false')",
		"; update tablename set code='javascript code' where 1--",
		"AND @@version like '5.0%', ",
		"/*!40110 and 1=0*/",
		"AND 1=0 UNION SELECT DATABASE()",
		"load_file('filename')",
		"limit 1 into outfile 'aaa'",
		"OR IF(1, BENCHMARK(#ofcicies, action_to_be_performed), 'false')",
		"AND 1=CONVERT(int, db_name())",

		// PostgresSQL
		"and 1::int=1",
	} {
		req, err := http.NewRequest(http.MethodPost, "https://example.com/?id=1 "+url.QueryEscape(id), nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("User-Agent", testUserAgent)
		result, err := template.MatchRequest(requests.NewTestRequest(req), nil, firewallconfigs.ServerCaptchaTypeNone)
		if err != nil {
			t.Fatal(err)
		}
		a.IsNotNil(result.Set)
		if result.Set != nil {
			a.IsTrue(lists.ContainsAny([]string{"7010"}, result.Set.Code))
		} else {
			t.Log("break:", id)
		}
	}
}

func TestTemplateSQLInjection(t *testing.T) {
	template, err := waf.Template()
	if err != nil {
		t.Fatal(err)
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
	req.Header.Set("User-Agent", testUserAgent)
	_, _, result, err := group.MatchRequest(requests.NewTestRequest(req))
	if err != nil {
		t.Fatal(err)
	}
	if result != nil {
		t.Log(result)
	}
}

func BenchmarkTemplateSQLInjection(b *testing.B) {
	template, err := waf.Template()
	if err != nil {
		b.Fatal(err)
	}
	var group = template.FindRuleGroupWithCode("sqlInjection")
	if group == nil {
		b.Fatal("group not found")
		return
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, err := http.NewRequest(http.MethodPost, "https://example.com/?id=1234"+types.String(rand.Int()%10000), nil)
			if err != nil {
				b.Fatal(err)
			}
			req.Header.Set("User-Agent", testUserAgent)

			_, _, result, err := group.MatchRequest(requests.NewTestRequest(req))
			if err != nil {
				b.Fatal(err)
			}
			_ = result
		}
	})
}

func testTemplate20001(a *assert.Assertion, t *testing.T, template *waf.WAF) {
	// enable bot rule set
	for _, g := range template.Inbound {
		if g.Code == "bot" {
			g.IsOn = true
			break
		}
	}

	for _, bot := range []string{
		"Googlebot",
		"AdsBot-Google",
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
		result, err := template.MatchRequest(requests.NewTestRequest(req), nil, firewallconfigs.ServerCaptchaTypeNone)
		if err != nil {
			t.Fatal(err)
		}
		a.IsNotNil(result.Set)
		if result.Set != nil {
			a.IsTrue(lists.ContainsAny([]string{"20001"}, result.Set.Code))
		} else {
			t.Log("break:", bot)
		}
	}
}

func BenchmarkTemplatePathTraversal(b *testing.B) {
	runtime.GOMAXPROCS(4)

	template, err := waf.Template()
	if err != nil {
		b.Fatal(err)
	}
	var group = template.FindRuleGroupWithCode("pathTraversal")
	if group == nil {
		b.Fatal("group not found")
		return
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, err := http.NewRequest(http.MethodPost, "https://example.com/?id=1234"+types.String(rand.Int()%10000)+"&name=lily&time=12345678910", nil)
			if err != nil {
				b.Fatal(err)
			}
			req.Header.Set("User-Agent", testUserAgent)

			_, _, result, err := group.MatchRequest(requests.NewTestRequest(req))
			if err != nil {
				b.Fatal(err)
			}
			_ = result
		}
	})
}

func BenchmarkTemplateCC2(b *testing.B) {
	runtime.GOMAXPROCS(4)

	template, err := waf.Template()
	if err != nil {
		b.Fatal(err)
	}
	var group = template.FindRuleGroupWithCode("cc2")
	if group == nil {
		b.Fatal("group not found")
		return
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, err := http.NewRequest(http.MethodPost, "https://example.com/?id=1234"+types.String(rand.Int()%10000)+"&name=lily&time=12345678910", nil)
			if err != nil {
				b.Fatal(err)
			}
			req.Header.Set("User-Agent", testUserAgent)

			_, _, result, err := group.MatchRequest(requests.NewTestRequest(req))
			if err != nil {
				b.Fatal(err)
			}
			_ = result
		}
	})
}
