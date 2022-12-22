// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package agents_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils/agents"
	"testing"
)

func TestIsAgentFromUserAgent(t *testing.T) {
	t.Log(agents.IsAgentFromUserAgent("Mozilla/5.0 (Linux;u;Android 4.2.2;zh-cn;) AppleWebKit/534.46 (KHTML,like Gecko) Version/5.1 Mobile Safari/10600.6.3 (compatible; Baiduspider/2.0; +http://www.baidu.com/search/spider.html)"))
	t.Log(agents.IsAgentFromUserAgent("Mozilla/5.0 (Linux;u;Android 4.2.2;zh-cn;)"))
}

func BenchmarkIsAgentFromUserAgent(b *testing.B) {
	for i := 0; i < b.N; i++ {
		agents.IsAgentFromUserAgent("Mozilla/5.0 (Linux;u;Android 4.2.2;zh-cn;) AppleWebKit/534.46 (KHTML,like Gecko) Version/5.1 Mobile Safari/10600.6.3 (compatible; Yaho)")
	}
}
