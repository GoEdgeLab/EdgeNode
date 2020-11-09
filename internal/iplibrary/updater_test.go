package iplibrary

import (
	_ "github.com/iwind/TeaGo/bootstrap"
	"github.com/iwind/TeaGo/dbs"
	"testing"
)

func TestUpdater_loop(t *testing.T) {
	dbs.NotifyReady()

	updater := NewUpdater()
	err := updater.loop()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ok")
}
