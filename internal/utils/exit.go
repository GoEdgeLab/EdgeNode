// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package utils

import (
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"os"
)

func Exit() {
	events.Notify(events.EventTerminated)
	os.Exit(0)
}
