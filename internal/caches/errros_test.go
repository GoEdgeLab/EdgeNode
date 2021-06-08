// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package caches

import (
	"github.com/iwind/TeaGo/assert"
	"testing"
)

func TestCanIgnoreErr(t *testing.T) {
	a := assert.NewAssertion(t)

	a.IsTrue(CanIgnoreErr(ErrFileIsWriting))
	a.IsTrue(CanIgnoreErr(NewCapacityError("over capcity")))
	a.IsFalse(CanIgnoreErr(ErrNotFound))
}
