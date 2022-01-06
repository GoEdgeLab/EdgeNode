// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package iplibrary

import "testing"

func TestNewProviderManager(t *testing.T) {
	var manager = NewProviderManager()
	err := manager.loop()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(manager.Lookup("阿里云"))
	t.Log(manager.Lookup("阿里云2"))
}
