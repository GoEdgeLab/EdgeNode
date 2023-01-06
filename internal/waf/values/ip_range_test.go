// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package values_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/values"
	"github.com/iwind/TeaGo/assert"
	"github.com/iwind/TeaGo/logs"
	"testing"
)

func TestIPRange_ParseIPRangeList(t *testing.T) {
	{
		var r = values.ParseIPRangeList("")
		logs.PrintAsJSON(r, t)
	}
	{
		var r = values.ParseIPRangeList("192.168.2.1")
		logs.PrintAsJSON(r, t)
	}
	{
		var r = values.ParseIPRangeList(`192.168.2.1  
192.168.1.100/24
192.168.1.1-192.168.2.100
192.168.1.2,192.168.2.200
192.168.2.200 - 192.168.2.100
# 以下是错误的
192.168
192.168.100.1-1`)
		logs.PrintAsJSON(r, t)
	}
}

func TestIPRange_Contains(t *testing.T) {
	{
		var a = assert.NewAssertion(t)
		var r = values.ParseIPRangeList(`192.168.2.1  
192.168.1.100/24
192.168.1.1-192.168.2.100
192.168.2.2,192.168.2.200
192.168.3.200 - 192.168.3.100
192.168.4.100
192.168.5.1/26`)
		a.IsTrue(r.Contains("192.168.1.102"))
		a.IsTrue(r.Contains("192.168.2.101"))
		a.IsTrue(r.Contains("192.168.1.1"))
		a.IsTrue(r.Contains("192.168.2.100"))
		a.IsFalse(r.Contains("192.168.2.201"))
		a.IsTrue(r.Contains("192.168.3.101"))
		a.IsTrue(r.Contains("192.168.4.100"))
		a.IsTrue(r.Contains("192.168.5.63"))
		a.IsFalse(r.Contains("192.168.5.128"))
	}
}
