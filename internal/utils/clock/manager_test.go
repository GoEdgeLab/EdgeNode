// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package clock_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils/clock"
	"github.com/TeaOSLab/EdgeNode/internal/utils/testutils"
	"testing"
)

func TestReadServer(t *testing.T) {
	if !testutils.IsSingleTesting() {
		return
	}

	t.Log(clock.NewClockManager().ReadServer("pool.ntp.org"))
}
