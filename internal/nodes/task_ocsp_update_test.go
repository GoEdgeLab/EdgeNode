// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package nodes_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/nodes"
	"testing"
)

func TestOCSPUpdateTask_Loop(t *testing.T) {
	var task = &nodes.OCSPUpdateTask{}
	err := task.Loop()
	if err != nil {
		t.Fatal(err)
	}
}
