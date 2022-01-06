// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package iplibrary

import "testing"

func TestNewCityManager(t *testing.T) {
	var manager = NewCityManager()
	err := manager.loop()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(manager.Lookup(16, "许昌市"))
}
