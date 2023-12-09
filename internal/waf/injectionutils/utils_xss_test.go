// Copyright 2023 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package injectionutils_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/injectionutils"
	"github.com/TeaOSLab/EdgeNode/internal/waf/utils"
	"github.com/iwind/TeaGo/assert"
	"runtime"
	"testing"
)

func TestDetectXSS(t *testing.T) {
	var a = assert.NewAssertion(t)
	a.IsFalse(injectionutils.DetectXSS(""))
	a.IsFalse(injectionutils.DetectXSS("abc"))
	a.IsTrue(injectionutils.DetectXSS("<script>"))
	a.IsTrue(injectionutils.DetectXSS("<link>"))
	a.IsFalse(injectionutils.DetectXSS("<html><span>"))
	a.IsFalse(injectionutils.DetectXSS("&lt;script&gt;"))
	a.IsTrue(injectionutils.DetectXSS("/path?onmousedown=a"))
	a.IsTrue(injectionutils.DetectXSS("/path?onkeyup=a"))
	a.IsTrue(injectionutils.DetectXSS("onkeyup=a"))
	a.IsTrue(injectionutils.DetectXSS("<iframe scrolling='no'>"))
	a.IsFalse(injectionutils.DetectXSS("<html><body><span>RequestId: 1234567890</span></body></html>"))
}

func BenchmarkDetectXSS_MISS(b *testing.B) {
	var result = injectionutils.DetectXSS("<html><body><span>RequestId: 1234567890</span></body></html>")
	if result {
		b.Fatal("'result' should not be 'true'")
	}

	runtime.GOMAXPROCS(4)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = injectionutils.DetectXSS("<html><body><span>RequestId: 1234567890</span></body></html>")
		}
	})
}

func BenchmarkDetectXSS_MISS_Cache(b *testing.B) {
	var result = injectionutils.DetectXSS("<html><body><span>RequestId: 1234567890</span></body></html>")
	if result {
		b.Fatal("'result' should not be 'true'")
	}

	runtime.GOMAXPROCS(4)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = injectionutils.DetectXSSCache("<html><body><span>RequestId: 1234567890</span></body></html>", utils.CacheMiddleLife)
		}
	})
}

func BenchmarkDetectXSS_HIT(b *testing.B) {
	var result = injectionutils.DetectXSS("<html><body><span>RequestId: 1234567890</span><script src=\"\"></script></body></html>")
	if !result {
		b.Fatal("'result' should not be 'false'")
	}

	runtime.GOMAXPROCS(4)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = injectionutils.DetectXSS("<html><body><span>RequestId: 1234567890</span><script src=\"\"></script></body></html>")
		}
	})
}
