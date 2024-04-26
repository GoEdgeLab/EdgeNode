// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package bfs_test

import (
	"bytes"
	"github.com/TeaOSLab/EdgeNode/internal/utils/bfs"
	"github.com/TeaOSLab/EdgeNode/internal/utils/fasttime"
	"github.com/TeaOSLab/EdgeNode/internal/utils/testutils"
	"github.com/iwind/TeaGo/logs"
	"net/http"
	"testing"
	"time"
)

func TestNewFileWriter(t *testing.T) {
	bFile, err := bfs.NewBlocksFile("testdata/test.b", bfs.DefaultBlockFileOptions)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		if !testutils.IsSingleTesting() {
			_ = bFile.RemoveAll()
		} else {
			_ = bFile.Close()
		}
	}()

	writer, err := bFile.OpenFileWriter(bfs.Hash("123456"), -1, false)
	if err != nil {
		t.Fatal(err)
	}

	err = writer.WriteMeta(http.StatusOK, fasttime.Now().Unix()+3600, -1)
	if err != nil {
		t.Fatal(err)
	}

	_, err = writer.WriteHeader([]byte("Content-Type: text/html; charset=utf-8"))
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 3; i++ {
		n, writeErr := writer.WriteBody([]byte("Hello,World"))
		if writeErr != nil {
			t.Fatal(writeErr)
		}

		t.Log("wrote:", n, "bytes")
	}

	err = writer.Close()
	if err != nil {
		t.Fatal(err)
	}
}

func TestNewFileWriter_LargeFile(t *testing.T) {
	bFile, err := bfs.NewBlocksFile("testdata/test.b", bfs.DefaultBlockFileOptions)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		if !testutils.IsSingleTesting() {
			_ = bFile.RemoveAll()
		} else {
			_ = bFile.Close()
		}
	}()

	writer, err := bFile.OpenFileWriter(bfs.Hash("123456@LARGE"), -1, false)
	if err != nil {
		t.Fatal(err)
	}

	err = writer.WriteMeta(http.StatusOK, fasttime.Now().Unix()+86400, -1)
	if err != nil {
		t.Fatal(err)
	}

	var countBlocks = 1 << 10
	if !testutils.IsSingleTesting() {
		countBlocks = 2
	}

	var data = bytes.Repeat([]byte{'A'}, 16<<10)

	var before = time.Now()
	for i := 0; i < countBlocks; i++ {
		_, err = writer.WriteBody(data)
		if err != nil {
			t.Fatal(err)
		}
	}

	err = writer.Close()
	if err != nil {
		t.Fatal(err)
	}

	logs.Println("cost:", time.Since(before).Seconds()*1000, "ms")
}

func TestFileWriter_WriteBodyAt(t *testing.T) {
	bFile, err := bfs.NewBlocksFile("testdata/test.b", bfs.DefaultBlockFileOptions)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		if !testutils.IsSingleTesting() {
			_ = bFile.RemoveAll()
		} else {
			_ = bFile.Close()
		}
	}()

	writer, err := bFile.OpenFileWriter(bfs.Hash("123456"), 1<<20, true)
	if err != nil {
		t.Fatal(err)
	}

	{
		n, writeErr := writer.WriteBodyAt([]byte("Hello,World"), 1024)
		if writeErr != nil {
			t.Fatal(writeErr)
		}

		t.Log("wrote:", n, "bytes")
	}
}
