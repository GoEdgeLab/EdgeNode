// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package bfs_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils/bfs"
	"github.com/TeaOSLab/EdgeNode/internal/utils/fasttime"
	"github.com/TeaOSLab/EdgeNode/internal/utils/testutils"
	"github.com/iwind/TeaGo/logs"
	"sync"
	"testing"
	"time"
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

func TestNewMetaFile_Large(t *testing.T) {
	var count = 2

	if testutils.IsSingleTesting() {
		count = 100
	}

	var before = time.Now()
	for i := 0; i < count; i++ {
		mFile, err := bfs.OpenMetaFile("testdata/test2.m", &sync.RWMutex{})
		if err != nil {
			if bfs.IsNotExist(err) {
				continue
			}
			t.Fatal(err)
		}
		_ = mFile.Close()
	}
	var costMs = time.Since(before).Seconds() * 1000
	t.Logf("cost: %.2fms, qps: %.2fms/file", costMs, costMs/float64(count))
}

func TestNewMetaFile_Memory(t *testing.T) {
	var count = 2

	if testutils.IsSingleTesting() {
		count = 100
	}

	var stat1 = testutils.ReadMemoryStat()

	var mFiles []*bfs.MetaFile

	for i := 0; i < count; i++ {
		mFile, err := bfs.OpenMetaFile("testdata/test2.m", &sync.RWMutex{})
		if err != nil {
			if bfs.IsNotExist(err) {
				continue
			}
			t.Fatal(err)
		}

		_ = mFile.Close()
		mFiles = append(mFiles, mFile)
	}

	var stat2 = testutils.ReadMemoryStat()
	t.Log((stat2.HeapInuse-stat1.HeapInuse)>>20, "MiB")
}

func TestMetaFile_FileHeaders(t *testing.T) {
	mFile, openErr := bfs.OpenMetaFile("testdata/test2.m", &sync.RWMutex{})
	if openErr != nil {
		if bfs.IsNotExist(openErr) {
			return
		}
		t.Fatal(openErr)
	}
	_ = mFile.Close()
	for hash, lazyHeader := range mFile.FileHeaders() {
		header, err := lazyHeader.FileHeaderUnsafe()
		if err != nil {
			t.Fatal(err)
		}
		t.Log(hash, header.ModifiedAt, header.BodySize)
	}
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
