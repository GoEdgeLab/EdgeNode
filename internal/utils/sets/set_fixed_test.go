// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package setutils_test

import (
	setutils "github.com/TeaOSLab/EdgeNode/internal/utils/sets"
	"github.com/iwind/TeaGo/assert"
	"github.com/iwind/TeaGo/rands"
	"testing"
)

func TestNewFixedSet(t *testing.T) {
	var a = assert.NewAssertion(t)

	{
		var set = setutils.NewFixedSet(0)
		set.Push(1)
		set.Push(2)
		set.Push(2)
		a.IsTrue(set.Size() == 2)
		a.IsTrue(set.Has(1))
		a.IsTrue(set.Has(2))
	}

	{
		var set = setutils.NewFixedSet(1)
		set.Push(1)
		set.Push(2)
		set.Push(3)
		a.IsTrue(set.Size() == 1)
		a.IsFalse(set.Has(1))
		a.IsTrue(set.Has(3))
		a.IsFalse(set.Has(4))
	}
}

func TestFixedSet_Reset(t *testing.T) {
	var a = assert.NewAssertion(t)

	var set = setutils.NewFixedSet(3)
	set.Push(1)
	set.Push(2)
	set.Push(3)
	set.Reset()
	a.IsTrue(set.Size() == 0)
}

func BenchmarkFixedSet_Has(b *testing.B) {
	var count = 1_000_000
	var set = setutils.NewFixedSet(count)
	for i := 0; i < count; i++ {
		set.Push(i)
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			set.Has(rands.Int(0, 100_000))
		}
	})
}
