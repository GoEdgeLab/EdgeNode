package iplibrary

import "testing"

func TestIPListManager_loop(t *testing.T) {
	manager := NewIPListManager()
	manager.pageSize = 2
	err := manager.loop()
	if err != nil {
		t.Fatal(err)
	}
}
