// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package agents_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils/agents"
	"github.com/iwind/TeaGo/Tea"
	_ "github.com/iwind/TeaGo/bootstrap"
	"testing"
)

func TestNewManager(t *testing.T) {
	var db = agents.NewDB(Tea.Root + "/data/agents.db")
	err := db.Init()
	if err != nil {
		t.Fatal(err)
	}

	var manager = agents.NewManager()
	manager.SetDB(db)
	err = manager.Load()
	if err != nil {
		t.Fatal(err)
	}

	_, err = manager.Loop()
	if err != nil {
		t.Fatal(err)
	}

	t.Log(manager.LookupIP("192.168.3.100"))
}
