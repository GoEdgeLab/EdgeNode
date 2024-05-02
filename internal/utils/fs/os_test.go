// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package fsutils_test

import (
	fsutils "github.com/TeaOSLab/EdgeNode/internal/utils/fs"
	"github.com/iwind/TeaGo/assert"
	"os"
	"testing"
)

func TestOpenFile(t *testing.T) {
	f, err := fsutils.OpenFile("./os_test.go", os.O_RDONLY, 0444)
	if err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
}

func TestExistFile(t *testing.T) {
	var a = assert.NewAssertion(t)

	{
		b, err := fsutils.ExistFile("./os_test.go")
		if err != nil {
			t.Fatal(err)
		}
		a.IsTrue(b)
	}

	{
		b, err := fsutils.ExistFile("./os_test2.go")
		if err != nil {
			t.Fatal(err)
		}
		a.IsFalse(b)
	}
}
