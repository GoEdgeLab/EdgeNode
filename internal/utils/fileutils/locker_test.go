// Copyright 2023 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package fileutils_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils/fileutils"
	"testing"
)

func TestLocker_Lock(t *testing.T) {
	var path = "/tmp/file-test"
	var locker = fileutils.NewLocker(path)
	err := locker.Lock()
	if err != nil {
		t.Fatal(err)
	}
	_ = locker.Release()

	var locker2 = fileutils.NewLocker(path)
	err = locker2.Lock()
	if err != nil {
		t.Fatal(err)
	}
}
