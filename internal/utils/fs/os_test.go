// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package fsutils_test

import (
	fsutils "github.com/TeaOSLab/EdgeNode/internal/utils/fs"
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
