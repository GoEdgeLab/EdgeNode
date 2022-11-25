// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package jsonutils_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils/jsonutils"
	"github.com/iwind/TeaGo/assert"
	"github.com/iwind/TeaGo/maps"
	"testing"
)

func TestEqual(t *testing.T) {
	var a = assert.NewAssertion(t)

	{
		var m1 = maps.Map{"a": 1, "b2": true}
		var m2 = maps.Map{"b2": true, "a": 1}
		a.IsTrue(jsonutils.Equal(m1, m2))
	}

	{
		var m1 = maps.Map{"a": 1, "b2": true, "c": nil}
		var m2 = maps.Map{"b2": true, "a": 1}
		a.IsFalse(jsonutils.Equal(m1, m2))
	}
}
