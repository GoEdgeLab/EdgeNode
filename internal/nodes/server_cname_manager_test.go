// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package nodes

import (
	"testing"
	"time"
)

func TestServerCNameManager_Lookup(t *testing.T) {
	var cnameManager = NewServerCNAMEManager()
	t.Log(cnameManager.Lookup("www.yun4s.cn"))

	var before = time.Now()
	defer func() {
		t.Log(time.Since(before).Seconds()*1000, "ms")
	}()
	t.Log(cnameManager.Lookup("www.yun4s.cn"))
}
