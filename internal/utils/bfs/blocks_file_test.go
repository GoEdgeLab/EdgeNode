// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package bfs_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils/bfs"
	"os"
	"testing"
)

func TestBlocksFile_OpenFileWriter_SameHash(t *testing.T) {
	bFile, openErr := bfs.OpenBlocksFile("testdata/test.b", bfs.DefaultBlockFileOptions)
	if openErr != nil {
		if os.IsNotExist(openErr) {
			return
		}
		t.Fatal(openErr)
	}

	{
		writer, err := bFile.OpenFileWriter(bfs.Hash("123456"), -1, false)
		if err != nil {
			t.Fatal(err)
		}
		_ = writer.Close()
	}

	{
		writer, err := bFile.OpenFileWriter(bfs.Hash("123456"), -1, false)
		if err != nil {
			t.Fatal(err)
		}
		_ = writer.Close()
	}
}

func TestBlocksFile_RemoveAll(t *testing.T) {
	bFile, err := bfs.OpenBlocksFile("testdata/test.b", bfs.DefaultBlockFileOptions)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		t.Fatal(err)
	}
	defer func() {
		_ = bFile.Close()
	}()

	err = bFile.RemoveAll()
	if err != nil {
		t.Fatal(err)
	}
}
