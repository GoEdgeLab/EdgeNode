// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package bfs_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils/bfs"
	"github.com/TeaOSLab/EdgeNode/internal/utils/fasttime"
	"github.com/iwind/TeaGo/Tea"
	_ "github.com/iwind/TeaGo/bootstrap"
	"github.com/iwind/TeaGo/logs"
	"io"
	"testing"
)

func TestFS_OpenFileWriter(t *testing.T) {
	fs, openErr := bfs.OpenFS(Tea.Root+"/data/bfs/test", bfs.DefaultFSOptions)
	if openErr != nil {
		t.Fatal(openErr)
	}
	defer func() {
		_ = fs.Close()
	}()

	{
		writer, err := fs.OpenFileWriter(bfs.Hash("123456"), -1, false)
		if err != nil {
			t.Fatal(err)
		}

		err = writer.WriteMeta(200, fasttime.Now().Unix()+3600, -1)
		if err != nil {
			t.Fatal(err)
		}

		_, err = writer.WriteBody([]byte("Hello, World"))
		if err != nil {
			t.Fatal(err)
		}

		err = writer.Close()
		if err != nil {
			t.Fatal(err)
		}
	}

	{
		writer, err := fs.OpenFileWriter(bfs.Hash("654321"), 100, true)
		if err != nil {
			t.Fatal(err)
		}

		_, err = writer.WriteBody([]byte("Hello, World"))
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestFS_OpenFileReader(t *testing.T) {
	fs, openErr := bfs.OpenFS(Tea.Root+"/data/bfs/test", bfs.DefaultFSOptions)
	if openErr != nil {
		t.Fatal(openErr)
	}
	defer func() {
		_ = fs.Close()
	}()

	reader, err := fs.OpenFileReader(bfs.Hash("123456"), false)
	if err != nil {
		if bfs.IsNotExist(err) {
			t.Log(err)
			return
		}
		t.Fatal(err)
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(data))
	logs.PrintAsJSON(reader.FileHeader(), t)
}

func TestFS_ExistFile(t *testing.T) {
	fs, openErr := bfs.OpenFS(Tea.Root+"/data/bfs/test", bfs.DefaultFSOptions)
	if openErr != nil {
		t.Fatal(openErr)
	}
	defer func() {
		_ = fs.Close()
	}()

	exist, err := fs.ExistFile(bfs.Hash("123456"))
	if err != nil {
		t.Fatal(err)
	}
	t.Log("exist:", exist)
}

func TestFS_RemoveFile(t *testing.T) {
	fs, openErr := bfs.OpenFS(Tea.Root+"/data/bfs/test", bfs.DefaultFSOptions)
	if openErr != nil {
		t.Fatal(openErr)
	}
	defer func() {
		_ = fs.Close()
	}()

	var hash = bfs.Hash("123456")
	err := fs.RemoveFile(hash)
	if err != nil {
		t.Fatal(err)
	}

	exist, err := fs.ExistFile(bfs.Hash("123456"))
	if err != nil {
		t.Fatal(err)
	}
	t.Log("exist:", exist)
}
