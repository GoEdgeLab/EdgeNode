// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package nodes

import (
	_ "github.com/iwind/TeaGo/bootstrap"
	"testing"
)

func TestUpgradeManager_install(t *testing.T) {
	err := NewUpgradeManager().install()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ok")
}
