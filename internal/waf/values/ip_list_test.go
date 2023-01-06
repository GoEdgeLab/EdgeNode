// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package values_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/values"
	"github.com/iwind/TeaGo/assert"
	"testing"
)

func TestParseIPList(t *testing.T) {
	var a = assert.NewAssertion(t)

	{
		var list = values.ParseIPList("")
		a.IsFalse(list.Contains("192.168.1.100"))
	}

	{
		var list = values.ParseIPList(`
192.168.1.1
192.168.1.101`)
		a.IsFalse(list.Contains("192.168.1.100"))
		a.IsTrue(list.Contains("192.168.1.101"))
	}
}
