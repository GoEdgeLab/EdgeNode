// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package iplibrary_test

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeNode/internal/iplibrary"
	"github.com/TeaOSLab/EdgeNode/internal/utils/fasttime"
	"github.com/TeaOSLab/EdgeNode/internal/utils/testutils"
	"github.com/TeaOSLab/EdgeNode/internal/zero"
	"testing"
	"time"
)

func TestKVIPList_AddItem(t *testing.T) {
	kv, err := iplibrary.NewKVIPList()
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		_ = kv.Flush()
	}()

	{
		err = kv.AddItem(&pb.IPItem{
			Id:        1,
			IpFrom:    "192.168.1.101",
			IpTo:      "",
			Version:   1,
			ExpiredAt: fasttime.NewFastTime().Unix() + 60,
			ListId:    1,
			IsDeleted: false,
			ListType:  "white",
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	{
		err = kv.AddItem(&pb.IPItem{
			Id:        2,
			IpFrom:    "192.168.1.102",
			IpTo:      "",
			Version:   2,
			ExpiredAt: fasttime.NewFastTime().Unix() + 60,
			ListId:    1,
			IsDeleted: false,
			ListType:  "white",
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	{
		err = kv.AddItem(&pb.IPItem{
			Id:        3,
			IpFrom:    "192.168.1.103",
			IpTo:      "",
			Version:   3,
			ExpiredAt: fasttime.NewFastTime().Unix() + 60,
			ListId:    1,
			IsDeleted: false,
			ListType:  "white",
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestKVIPList_AddItems_Many(t *testing.T) {
	kv, err := iplibrary.NewKVIPList()
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		_ = kv.Flush()
	}()

	var count = 2
	var from = 1
	if testutils.IsSingleTesting() {
		count = 2_000_000
	}

	var before = time.Now()
	defer func() {
		t.Logf("cost: %.2f s", time.Since(before).Seconds())
	}()

	for i := from; i <= from+count; i++ {
		err = kv.AddItem(&pb.IPItem{
			Id:        int64(i),
			IpFrom:    testutils.RandIP(),
			IpTo:      "",
			Version:   int64(i),
			ExpiredAt: fasttime.NewFastTime().Unix() + 86400,
			ListId:    1,
			IsDeleted: false,
			ListType:  "white",
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestKVIPList_DeleteExpiredItems(t *testing.T) {
	kv, err := iplibrary.NewKVIPList()
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		_ = kv.Flush()
	}()

	err = kv.DeleteExpiredItems()
	if err != nil {
		t.Fatal(err)
	}
}

func TestKVIPList_UpdateMaxVersion(t *testing.T) {
	kv, err := iplibrary.NewKVIPList()
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		_ = kv.Flush()
	}()

	err = kv.UpdateMaxVersion(101)
	if err != nil {
		t.Fatal(err)
	}

	maxVersion, err := kv.ReadMaxVersion()
	if err != nil {
		t.Fatal(err)
	}

	t.Log("version:", maxVersion)
}

func TestKVIPList_ReadMaxVersion(t *testing.T) {
	kv, err := iplibrary.NewKVIPList()
	if err != nil {
		t.Fatal(err)
	}

	maxVersion, err := kv.ReadMaxVersion()
	if err != nil {
		t.Fatal(err)
	}

	t.Log("version:", maxVersion)
}

func TestKVIPList_ReadItems(t *testing.T) {
	kv, err := iplibrary.NewKVIPList()
	if err != nil {
		t.Fatal(err)
	}

	for {
		items, goNext, readErr := kv.ReadItems(0, 2)
		if readErr != nil {
			t.Fatal(readErr)
		}
		t.Log("====")
		for _, item := range items {
			t.Log(item.Id)
		}

		if !goNext {
			break
		}
	}
}

func TestKVIPList_CountItems(t *testing.T) {
	kv, err := iplibrary.NewKVIPList()
	if err != nil {
		t.Fatal(err)
	}

	var count int
	var m = map[int64]zero.Zero{}
	for {
		items, goNext, readErr := kv.ReadItems(0, 1000)
		if readErr != nil {
			t.Fatal(readErr)
		}
		for _, item := range items {
			count++
			m[item.Id] = zero.Zero{}
		}

		if !goNext {
			break
		}
	}
	t.Log("count:", count, "len:", len(m))
}

func TestKVIPList_Inspect(t *testing.T) {
	kv, err := iplibrary.NewKVIPList()
	if err != nil {
		t.Fatal(err)
	}
	err = kv.TestInspect(t)
	if err != nil {
		t.Fatal(err)
	}
}
