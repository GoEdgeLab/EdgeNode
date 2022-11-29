// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package rangeutils_test

import (
	rangeutils "github.com/TeaOSLab/EdgeNode/internal/utils/ranges"
	"github.com/iwind/TeaGo/assert"
	"testing"
)

func TestRange(t *testing.T) {
	var a = assert.NewAssertion(t)

	var r = rangeutils.NewRange(1, 100)
	a.IsTrue(r.Start() == 1)
	a.IsTrue(r.End() == 100)
	t.Log("start:", r.Start(), "end:", r.End())
}

func TestRange_Convert(t *testing.T) {
	var a = assert.NewAssertion(t)

	{
		var r = rangeutils.NewRange(1, 100)
		newR, ok := r.Convert(200)
		a.IsTrue(ok)
		a.IsTrue(newR.Start() == 1)
		a.IsTrue(newR.End() == 100)
	}

	{
		var r = rangeutils.NewRange(1, 100)
		newR, ok := r.Convert(50)
		a.IsTrue(ok)
		a.IsTrue(newR.Start() == 1)
		a.IsTrue(newR.End() == 49)
	}

	{
		var r = rangeutils.NewRange(1, 100)
		_, ok := r.Convert(0)
		a.IsFalse(ok)
	}

	{
		var r = rangeutils.NewRange(-30, -1)
		newR, ok := r.Convert(50)
		a.IsTrue(ok)
		a.IsTrue(newR.Start() == 50-30)
		a.IsTrue(newR.End() == 49)
	}

	{
		var r = rangeutils.NewRange(1000, 100)
		_, ok := r.Convert(0)
		a.IsFalse(ok)
	}

	{
		var r = rangeutils.NewRange(50, 100)
		_, ok := r.Convert(49)
		a.IsFalse(ok)
	}
}

func TestRange_ComposeContentRangeHeader(t *testing.T) {
	var r = rangeutils.NewRange(1, 100)
	t.Log(r.ComposeContentRangeHeader("1000"))
}
