package iplibrary

import (
	"runtime"
	"testing"
)

func TestProvinceManager_load(t *testing.T) {
	manager := NewProvinceManager()
	err := manager.load()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ok", manager.provinceMap)
}

func TestProvinceManager_loop(t *testing.T) {
	manager := NewProvinceManager()
	err := manager.loop()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ok", manager.provinceMap)
}

func TestProvinceManager_loop_skip(t *testing.T) {
	manager := NewProvinceManager()
	for i := 0; i < 10; i++ {
		err := manager.loop()
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestProvinceManager_Lookup(t *testing.T) {
	manager := NewProvinceManager()
	err := manager.load()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(manager.Lookup("安徽省"), manager.Lookup("北京市"))
}

func BenchmarkProvinceManager_Lookup(b *testing.B) {
	runtime.GOMAXPROCS(1)

	manager := NewProvinceManager()
	err := manager.load()
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		_ = manager.Lookup("安徽省")
	}
}
