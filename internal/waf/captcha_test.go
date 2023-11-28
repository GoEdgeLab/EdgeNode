// Copyright 2023 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package waf_test

import (
	"bytes"
	"fmt"
	"github.com/TeaOSLab/EdgeNode/internal/utils/testutils"
	"github.com/dchest/captcha"
	"runtime"
	"testing"
	"time"
)

func TestCaptchaMemory(t *testing.T) {
	if !testutils.IsSingleTesting() {
		return
	}

	var stat1 = &runtime.MemStats{}
	runtime.ReadMemStats(stat1)

	var count = 5_000
	var before = time.Now()

	for i := 0; i < count; i++ {
		var id = captcha.NewLen(6)
		var writer = &bytes.Buffer{}
		err := captcha.WriteImage(writer, id, 200, 100)
		if err != nil {
			t.Fatal(err)
		}
		captcha.VerifyString(id, "abc")
	}

	var stat2 = &runtime.MemStats{}
	runtime.ReadMemStats(stat2)
	t.Log((stat2.HeapInuse-stat1.HeapInuse)>>20, "MB", fmt.Sprintf("%.0f QPS", float64(count)/time.Since(before).Seconds()))
}

func BenchmarkCaptcha_VerifyCode_100_50(b *testing.B) {
	runtime.GOMAXPROCS(4)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var id = captcha.NewLen(6)
			var writer = &bytes.Buffer{}
			err := captcha.WriteImage(writer, id, 100, 50)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkCaptcha_VerifyCode_200_100(b *testing.B) {
	runtime.GOMAXPROCS(4)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var id = captcha.NewLen(6)
			var writer = &bytes.Buffer{}
			err := captcha.WriteImage(writer, id, 200, 100)
			if err != nil {
				b.Fatal(err)
			}
			_ = id
		}
	})
}
