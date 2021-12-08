// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package goman

import (
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	New(func() {
		t.Log("Hello")

		t.Log(List())
	})

	time.Sleep(1 * time.Second)
	t.Log(List())

	time.Sleep(1 * time.Second)
}
