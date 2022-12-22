// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package agents

import "strings"

var AllAgents = []*Agent{
	NewAgent("baidu", []string{".baidu.com."}, nil, []string{"Baidu"}),
	NewAgent("google", []string{".googlebot.com."}, nil, []string{"Google"}),
	NewAgent("bing", []string{".search.msn.com."}, nil, []string{"bingbot"}),
	NewAgent("sogou", []string{".sogou.com."}, nil, []string{"Sogou"}),
	NewAgent("youdao", []string{".163.com."}, nil, []string{"Youdao"}),
	NewAgent("yahoo", []string{".yahoo.com."}, nil, []string{"Yahoo"}),
	NewAgent("bytedance", []string{".bytedance.com."}, nil, []string{"Bytespider"}),
	NewAgent("sm", []string{".sm.cn."}, nil, []string{"YisouSpider"}),
	NewAgent("yandex", []string{".yandex.com.", ".yndx.net."}, nil, []string{"Yandex"}),
	NewAgent("semrush", []string{".semrush.com."}, nil, []string{"SEMrush"}),
}

func IsAgentFromUserAgent(userAgent string) bool {
	for _, agent := range AllAgents {
		if len(agent.Keywords) > 0 {
			for _, keyword := range agent.Keywords {
				if strings.Contains(userAgent, keyword) {
					return true
				}
			}
		}
	}
	return false
}
