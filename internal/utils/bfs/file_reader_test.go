// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package bfs_test

import (
	"fmt"
	"github.com/TeaOSLab/EdgeNode/internal/utils/bfs"
	"io"
	"testing"
	"time"
)

func TestFileReader_Read_SmallBuf(t *testing.T) {
	bFile, err := bfs.NewBlocksFile("testdata/test.b", bfs.DefaultBlockFileOptions)
	if err != nil {
		t.Fatal(err)
	}

	reader, err := bFile.OpenFileReader(bfs.Hash("123456"), false)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		_ = reader.Close()
	}()

	var buf = make([]byte, 3)
	for {
		n, readErr := reader.Read(buf)
		if n > 0 {
			t.Log(string(buf[:n]))
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			t.Fatal(readErr)
		}
	}
}

func TestFileReader_Read_LargeBuff(t *testing.T) {
	bFile, err := bfs.NewBlocksFile("testdata/test.b", bfs.DefaultBlockFileOptions)
	if err != nil {
		t.Fatal(err)
	}

	reader, err := bFile.OpenFileReader(bfs.Hash("123456"), false)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		_ = reader.Close()
	}()

	var buf = make([]byte, 128)
	for {
		n, readErr := reader.Read(buf)
		if n > 0 {
			t.Log(string(buf[:n]))
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			t.Fatal(readErr)
		}
	}
}

func TestFileReader_Read_LargeFile(t *testing.T) {
	bFile, err := bfs.NewBlocksFile("testdata/test.b", bfs.DefaultBlockFileOptions)
	if err != nil {
		t.Fatal(err)
	}

	reader, err := bFile.OpenFileReader(bfs.Hash("123456@LARGE"), false)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		_ = reader.Close()
	}()

	var buf = make([]byte, 16<<10)
	var totalSize int64
	var before = time.Now()
	for {
		n, readErr := reader.Read(buf)
		if n > 0 {
			totalSize += int64(n)
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			t.Fatal(readErr)
		}
	}
	t.Log("totalSize:", totalSize>>20, "MiB", "cost:", fmt.Sprintf("%.4fms", time.Since(before).Seconds()*1000))
}

func TestFileReader_ReadAt(t *testing.T) {
	bFile, err := bfs.NewBlocksFile("testdata/test.b", bfs.DefaultBlockFileOptions)
	if err != nil {
		t.Fatal(err)
	}

	reader, err := bFile.OpenFileReader(bfs.Hash("123456"), false)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		_ = reader.Close()
	}()

	{
		var buf = make([]byte, 3)
		n, readErr := reader.ReadAt(buf, 0)
		if n > 0 {
			t.Log(string(buf[:n]))
		}
		if readErr != nil && readErr != io.EOF {
			t.Fatal(readErr)
		}
	}

	{
		var buf = make([]byte, 3)
		n, readErr := reader.ReadAt(buf, 3)
		if n > 0 {
			t.Log(string(buf[:n]))
		}
		if readErr != nil && readErr != io.EOF {
			t.Fatal(readErr)
		}
	}

	{
		var buf = make([]byte, 11)
		n, readErr := reader.ReadAt(buf, 3)
		if n > 0 {
			t.Log(string(buf[:n]))
		}
		if readErr != nil && readErr != io.EOF {
			t.Fatal(readErr)
		}
	}

	{
		var buf = make([]byte, 3)
		n, readErr := reader.ReadAt(buf, 11)
		if n > 0 {
			t.Log(string(buf[:n]))
		}
		if readErr != nil && readErr != io.EOF {
			t.Fatal(readErr)
		}
	}

	{
		var buf = make([]byte, 3)
		n, readErr := reader.ReadAt(buf, 1000)
		if n > 0 {
			t.Log(string(buf[:n]))
		} else {
			t.Log("EOF")
		}
		if readErr != nil && readErr != io.EOF {
			t.Fatal(readErr)
		}
	}
}
