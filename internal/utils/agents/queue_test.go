// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package agents_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils/agents"
	"github.com/iwind/TeaGo/assert"
	_ "github.com/iwind/TeaGo/bootstrap"
	"testing"
	"time"
)

func TestParseQueue_Process(t *testing.T) {
	var queue = agents.NewQueue()
	go queue.Start()
	time.Sleep(1 * time.Second)
	queue.Push("220.181.13.100")
	time.Sleep(1 * time.Second)
}

func TestParseQueue_ParseIP(t *testing.T) {
	var queue = agents.NewQueue()
	for _, ip := range []string{
		"192.168.1.100",
		"42.120.160.1",
		"42.236.10.98",
		"124.115.0.100",
	} {
		ptr, err := queue.ParseIP(ip)
		if err != nil {
			t.Log(ip, "=>", err)
			continue
		}
		t.Log(ip, "=>", ptr)
	}
}

func TestParseQueue_ParsePtr(t *testing.T) {
	var a = assert.NewAssertion(t)

	var queue = agents.NewQueue()
	for _, s := range [][]string{
		{"baiduspider-220-181-108-101.crawl.baidu.com.", "baidu"},
		{"crawl-66-249-71-219.googlebot.com.", "google"},
		{"msnbot-40-77-167-31.search.msn.com.", "bing"},
		{"sogouspider-49-7-20-129.crawl.sogou.com.", "sogou"},
		{"m13102.mail.163.com.", "youdao"},
		{"yeurosport.pat1.tc2.yahoo.com.", "yahoo"},
		{"shenmaspider-42-120-160-1.crawl.sm.cn.", "sm"},
		{"93-158-161-39.spider.yandex.com.", "yandex"},
		{"25.bl.bot.semrush.com.", "semrush"},
	} {
		a.IsTrue(queue.ParsePtr(s[0]) == s[1])
	}
}

func BenchmarkQueue_ParsePtr(b *testing.B) {
	var queue = agents.NewQueue()

	for i := 0; i < b.N; i++ {
		for _, s := range [][]string{
			{"baiduspider-220-181-108-101.crawl.baidu.com.", "baidu"},
			{"crawl-66-249-71-219.googlebot.com.", "google"},
			{"msnbot-40-77-167-31.search.msn.com.", "bing"},
			{"sogouspider-49-7-20-129.crawl.sogou.com.", "sogou"},
			{"m13102.mail.163.com.", "youdao"},
			{"yeurosport.pat1.tc2.yahoo.com.", "yahoo"},
			{"shenmaspider-42-120-160-1.crawl.sm.cn.", "sm"},
			{"93-158-161-39.spider.yandex.com.", "yandex"},
			{"93.158.164.218-red.dhcp.yndx.net.", "yandex"},
			{"25.bl.bot.semrush.com.", "semrush"},
		} {
			queue.ParsePtr(s[0])
		}
	}
}
