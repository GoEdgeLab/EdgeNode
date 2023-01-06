// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package values_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/values"
	"github.com/iwind/TeaGo/assert"
	"testing"
)

func TestParseNumberList(t *testing.T) {
	var a = assert.NewAssertion(t)

	{
		var list = values.ParseNumberList("")
		a.IsFalse(list.Contains(123))
	}

	{
		var list = values.ParseNumberList(`123
456

789.1234`)
		a.IsTrue(list.Contains(123))
		a.IsFalse(list.Contains(0))
		a.IsFalse(list.Contains(789.123))
		a.IsTrue(list.Contains(789.1234))
	}
}
