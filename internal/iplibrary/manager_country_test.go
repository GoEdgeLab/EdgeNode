package iplibrary

import (
	"runtime"
	"testing"
)

func TestCountryManager_load(t *testing.T) {
	manager := NewCountryManager()
	err := manager.load()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ok", manager.countryMap)
}

func TestCountryManager_loop(t *testing.T) {
	manager := NewCountryManager()
	err := manager.loop()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ok", manager.countryMap)
}

func TestCountryManager_loop_skip(t *testing.T) {
	manager := NewCountryManager()
	for i := 0; i < 10; i++ {
		err := manager.loop()
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestCountryManager_Lookup(t *testing.T) {
	manager := NewCountryManager()
	err := manager.load()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(manager.Lookup("中国"), manager.Lookup("美国 "))
}

func BenchmarkCountryManager_Lookup(b *testing.B) {
	runtime.GOMAXPROCS(1)

	manager := NewCountryManager()
	err := manager.load()
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		_ = manager.Lookup("中国")
	}
}
