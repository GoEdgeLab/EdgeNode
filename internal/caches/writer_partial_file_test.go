// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package caches_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/caches"
	"github.com/iwind/TeaGo/types"
	"os"
	"testing"
	"time"
)

func TestPartialFileWriter_Write(t *testing.T) {
	var path = "/tmp/test_partial.cache"
	_ = os.Remove(path)

	var reader = func() {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		t.Log("["+types.String(len(data))+"]", string(data))
	}

	fp, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		t.Fatal(err)
	}
	var ranges = caches.NewPartialRanges(0)
	var writer = caches.NewPartialFileWriter(fp, "test", time.Now().Unix()+86500, -1, -1, true, true, 0, ranges, func() {
		t.Log("end")
	})
	_, err = writer.WriteHeader([]byte("header"))
	if err != nil {
		t.Fatal(err)
	}

	// 移动位置
	err = writer.WriteAt(100, []byte("HELLO"))
	if err != nil {
		t.Fatal(err)
	}

	err = writer.Close()
	if err != nil {
		t.Fatal(err)
	}

	reader()
}
