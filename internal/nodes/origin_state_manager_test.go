// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package nodes

import "testing"

func TestOriginManager_Loop(t *testing.T) {
	var manager = NewOriginStateManager()
	err := manager.Loop()
	if err != nil {
		t.Fatal(err)
	}

	t.Log(manager.stateMap)
}
