// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package checkpoints

import (
	"net/http"
	"runtime"
	"sort"
	"strings"
	"testing"
)

func BenchmarkRequestHeadersCheckpoint_RequestValue(b *testing.B) {
	runtime.GOMAXPROCS(1)

	var header = http.Header{
		"Content-Type":    []string{"keep-alive"},
		"User-Agent":      []string{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/96.0.4664.110 Safari/537.36"},
		"Accept-Encoding": []string{"gzip, deflate, br"},
		"Referer":         []string{"https://goedge.cn/"},
	}

	for i := 0; i < b.N; i++ {
		var headers = []string{}
		for k, v := range header {
			for _, subV := range v {
				headers = append(headers, k+": "+subV)
			}
		}
		sort.Strings(headers)
		_ = strings.Join(headers, "\n")
	}
}
