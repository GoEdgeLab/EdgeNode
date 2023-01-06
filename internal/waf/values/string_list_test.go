// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package values_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/values"
	"github.com/iwind/TeaGo/assert"
	"testing"
)

func TestParseStringList(t *testing.T) {
	var a = assert.NewAssertion(t)

	{
		var list = values.ParseStringList("")
		a.IsFalse(list.Contains("hello"))
	}

	{
		var list = values.ParseStringList(`hello

world
hi

people`)
		a.IsTrue(list.Contains("hello"))
		a.IsFalse(list.Contains("hello1"))
		a.IsTrue(list.Contains("hi"))
	}
}
