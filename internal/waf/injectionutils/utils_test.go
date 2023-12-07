// Copyright 2023 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package injectionutils_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/injectionutils"
	"github.com/iwind/TeaGo/assert"
	"runtime"
	"testing"
)

func TestDetectSQLInjection(t *testing.T) {
	var a = assert.NewAssertion(t)
	a.IsTrue(injectionutils.DetectSQLInjection("' UNION SELECT * FROM myTable"))
	a.IsTrue(injectionutils.DetectSQLInjection("asdf asd ; -1' and 1=1 union/* foo */select load_file('/etc/passwd')--"))
	a.IsFalse(injectionutils.DetectSQLInjection("' UNION SELECT1 * FROM myTable"))
	a.IsFalse(injectionutils.DetectSQLInjection("1234"))
	a.IsFalse(injectionutils.DetectSQLInjection(""))
	a.IsTrue(injectionutils.DetectSQLInjection("id=123 OR 1=1&b=2"))
	a.IsFalse(injectionutils.DetectSQLInjection("?"))
	a.IsFalse(injectionutils.DetectSQLInjection("/hello?age=22"))
	a.IsTrue(injectionutils.DetectSQLInjection("/sql/injection?id=123 or 1=1"))
	a.IsTrue(injectionutils.DetectSQLInjection("/sql/injection?id=123%20or%201=1"))
}

func BenchmarkDetectSQLInjection(b *testing.B) {
	runtime.GOMAXPROCS(4)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = injectionutils.DetectSQLInjection("asdf asd ; -1' and 1=1 union/* foo */select load_file('/etc/passwd')--")
		}
	})
}

func BenchmarkDetectSQLInjection_URL(b *testing.B) {
	runtime.GOMAXPROCS(4)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = injectionutils.DetectSQLInjection("/sql/injection?id=123 or 1=1")
		}
	})
}


func BenchmarkDetectSQLInjection_URL_Unescape(b *testing.B) {
	runtime.GOMAXPROCS(4)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = injectionutils.DetectSQLInjection("/sql/injection?id=123%20or%201=1")
		}
	})
}
