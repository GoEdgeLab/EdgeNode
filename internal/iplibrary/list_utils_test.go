// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package iplibrary

import (
	"testing"
	"time"
)

func TestIPIsAllowed(t *testing.T) {
	manager := NewIPListManager()
	manager.init()

	var before = time.Now()
	defer func() {
		t.Log(time.Since(before).Seconds()*1000, "ms")
	}()
	t.Log(AllowIP("127.0.0.1", 0))
	t.Log(AllowIP("127.0.0.1", 23))
}
