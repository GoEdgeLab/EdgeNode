package iplibrary

import (
	_ "github.com/iwind/TeaGo/bootstrap"
	"github.com/iwind/TeaGo/dbs"
	"testing"
)

func TestManager_Load(t *testing.T) {
	dbs.NotifyReady()

	manager := NewManager()
	lib, err := manager.Load()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(lib.Lookup("1.2.3.4"))
	t.Log(lib.Lookup("2.3.4.5"))
	t.Log(lib.Lookup("200.200.200.200"))
	t.Log(lib.Lookup("202.106.0.20"))
}

func TestNewManager(t *testing.T) {
	dbs.NotifyReady()
	t.Log(SharedLibrary)
}
