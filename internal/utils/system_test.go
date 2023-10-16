// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package utils

import "testing"

func TestSystemMemoryGB(t *testing.T) {
	t.Log(SystemMemoryGB())
	t.Log(SystemMemoryGB())
	t.Log(SystemMemoryGB())
	t.Log(SystemMemoryBytes())
	t.Log(SystemMemoryBytes())
	t.Log(SystemMemoryBytes()>>30, "GB")
}
