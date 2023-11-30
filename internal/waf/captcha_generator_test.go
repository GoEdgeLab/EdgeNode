// Copyright 2023 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package waf_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils/testutils"
	"github.com/TeaOSLab/EdgeNode/internal/waf"
	"github.com/iwind/TeaGo/assert"
	"github.com/iwind/TeaGo/types"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestCaptchaGenerator_NewCaptcha(t *testing.T) {
	var a = assert.NewAssertion(t)

	var generator = waf.NewCaptchaGenerator()
	var captchaId = generator.NewCaptcha(6)
	t.Log("captchaId:", captchaId)

	var digits = generator.Get(captchaId)
	var s []string
	for _, digit := range digits {
		s = append(s, types.String(digit))
	}
	t.Log(strings.Join(s, " "))

	a.IsTrue(generator.Verify(captchaId, strings.Join(s, "")))
	a.IsFalse(generator.Verify(captchaId, strings.Join(s, "")))
}

func TestCaptchaGenerator_NewCaptcha_UTF8(t *testing.T) {
	var a = assert.NewAssertion(t)

	var generator = waf.NewCaptchaGenerator()
	var captchaId = generator.NewCaptcha(6)
	t.Log("captchaId:", captchaId)

	var digits = generator.Get(captchaId)
	var s []string
	for _, digit := range digits {
		s = append(s, types.String(digit))
	}
	t.Log(strings.Join(s, " "))

	a.IsFalse(generator.Verify(captchaId, "中文真的很长"))
}

func TestCaptchaGenerator_NewCaptcha_Memory(t *testing.T) {
	runtime.GC()

	var stat1 = &runtime.MemStats{}
	runtime.ReadMemStats(stat1)

	var generator = waf.NewCaptchaGenerator()
	for i := 0; i < 1_000_000; i++ {
		generator.NewCaptcha(6)
	}

	if testutils.IsSingleTesting() {
		time.Sleep(1 * time.Second)
	}

	runtime.GC()

	var stat2 = &runtime.MemStats{}
	runtime.ReadMemStats(stat2)

	t.Log((stat2.HeapInuse-stat1.HeapInuse)>>10, "KiB")

	_ = generator
}

func BenchmarkNewCaptchaGenerator(b *testing.B) {
	runtime.GOMAXPROCS(4)

	var generator = waf.NewCaptchaGenerator()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			generator.NewCaptcha(6)
		}
	})
}
