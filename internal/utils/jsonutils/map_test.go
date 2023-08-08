// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package jsonutils

import (
	"github.com/iwind/TeaGo/assert"
	"github.com/iwind/TeaGo/maps"
	"testing"
)

func TestMapToObject(t *testing.T) {
	a := assert.NewAssertion(t)

	type typeA struct {
		B int  `json:"b"`
		C bool `json:"c"`
	}

	{
		var obj = &typeA{B: 1, C: true}
		m, err := ObjectToMap(obj)
		if err != nil {
			t.Fatal(err)
		}
		PrintT(m, t)
		a.IsTrue(m.GetInt("b") == 1)
		a.IsTrue(m.GetBool("c") == true)
	}

	{
		var obj = &typeA{}
		err := MapToObject(maps.Map{
			"b": 1024,
			"c": true,
		}, obj)
		if err != nil {
			t.Fatal(err)
		}
		a.IsTrue(obj.B == 1024)
		a.IsTrue(obj.C == true)
		PrintT(obj, t)
	}
}
