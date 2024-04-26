// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package bfs_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils/bfs"
	"os"
	"testing"
)

func TestBlocksFile_RemoveAll(t *testing.T) {
	bFile, err := bfs.NewBlocksFile("testdata/test.b", bfs.DefaultBlockFileOptions)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		t.Fatal(err)
	}

	err = bFile.RemoveAll()
	if err != nil {
		t.Fatal(err)
	}
}
