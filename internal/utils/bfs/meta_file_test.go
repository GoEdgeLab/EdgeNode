// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package bfs_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils/bfs"
	"github.com/TeaOSLab/EdgeNode/internal/utils/fasttime"
	"github.com/iwind/TeaGo/logs"
	"sync"
	"testing"
)

func TestNewMetaFile(t *testing.T) {
	mFile, err := bfs.OpenMetaFile("testdata/test.m", &sync.RWMutex{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = mFile.Close()
	}()

	var header, _ = mFile.FileHeader(bfs.Hash("123456"))
	logs.PrintAsJSON(header, t)
	//logs.PrintAsJSON(mFile.Headers(), t)
}

func TestMetaFile_WriteMeta(t *testing.T) {
	mFile, err := bfs.OpenMetaFile("testdata/test.m", &sync.RWMutex{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = mFile.Close()
	}()

	var hash = bfs.Hash("123456")
	err = mFile.WriteMeta(hash, 200, fasttime.Now().Unix()+3600, -1)
	if err != nil {
		t.Fatal(err)
	}

	err = mFile.WriteHeaderBlockUnsafe(hash, 123, 223)
	if err != nil {
		t.Fatal(err)
	}

	err = mFile.WriteBodyBlockUnsafe(hash, 223, 323, 0, 100)
	if err != nil {
		t.Fatal(err)
	}

	err = mFile.WriteBodyBlockUnsafe(hash, 323, 423, 100, 200)
	if err != nil {
		t.Fatal(err)
	}

	err = mFile.WriteClose(hash, 100, 200)
	if err != nil {
		t.Fatal(err)
	}

	//logs.PrintAsJSON(mFile.Header(hash), t)
}

func TestMetaFile_Write(t *testing.T) {
	mFile, err := bfs.OpenMetaFile("testdata/test.m", &sync.RWMutex{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = mFile.Close()
	}()

	var hash = bfs.Hash("123456")

	err = mFile.WriteBodyBlockUnsafe(hash, 0, 100, 0, 100)
	if err != nil {
		t.Fatal(err)
	}

	err = mFile.WriteClose(hash, 0, 100)
	if err != nil {
		t.Fatal(err)
	}
}

func TestMetaFile_RemoveFile(t *testing.T) {
	mFile, err := bfs.OpenMetaFile("testdata/test.m", &sync.RWMutex{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = mFile.Close()
	}()

	err = mFile.RemoveFile(bfs.Hash("123456"))
	if err != nil {
		t.Fatal(err)
	}
}

func TestMetaFile_Compact(t *testing.T) {
	mFile, err := bfs.OpenMetaFile("testdata/test.m", &sync.RWMutex{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = mFile.Close()
	}()

	err = mFile.Compact()
	if err != nil {
		t.Fatal(err)
	}
}

func TestMetaFile_RemoveAll(t *testing.T) {
	mFile, err := bfs.OpenMetaFile("testdata/test.m", &sync.RWMutex{})
	if err != nil {
		t.Fatal(err)
	}
	err = mFile.RemoveAll()
	if err != nil {
		t.Fatal(err)
	}
}
