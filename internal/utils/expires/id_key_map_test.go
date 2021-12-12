// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package expires

import (
	"github.com/iwind/TeaGo/assert"
	"github.com/iwind/TeaGo/logs"
	"testing"
)

func TestNewIdKeyMap(t *testing.T) {
	var a = assert.NewAssertion(t)

	var m = NewIdKeyMap()
	m.Add(1, "1")
	m.Add(1, "2")
	m.Add(100, "100")
	logs.PrintAsJSON(m.idKeys, t)
	logs.PrintAsJSON(m.keyIds, t)

	{
		k, ok := m.Key(1)
		a.IsTrue(ok)
		a.IsTrue(k == "2")
	}

	{
		_, ok := m.Key(2)
		a.IsFalse(ok)
	}

	m.DeleteKey("2")

	{
		_, ok := m.Key(1)
		a.IsFalse(ok)
	}

	logs.PrintAsJSON(m.idKeys, t)
	logs.PrintAsJSON(m.keyIds, t)

	m.DeleteId(100)

	logs.PrintAsJSON(m.idKeys, t)
	logs.PrintAsJSON(m.keyIds, t)
}
