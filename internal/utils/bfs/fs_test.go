// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package bfs_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils/bfs"
	"github.com/TeaOSLab/EdgeNode/internal/utils/testutils"
	"github.com/iwind/TeaGo/Tea"
	_ "github.com/iwind/TeaGo/bootstrap"
	"testing"
	"time"
)

func TestFS_OpenFileWriter(t *testing.T) {
	var fs = bfs.NewFS(Tea.Root+"/data/bfs/test", bfs.DefaultFSOptions)
	defer func() {
		_ = fs.Close()
	}()

	{
		writer, err := fs.OpenFileWriter(bfs.Hash("123456"), 100, true)
		if err != nil {
			t.Fatal(err)
		}

		_, err = writer.WriteBody([]byte("Hello, World"))
		if err != nil {
			t.Fatal(err)
		}
	}

	{
		writer, err := fs.OpenFileWriter(bfs.Hash("123456"), 100, true)
		if err != nil {
			t.Fatal(err)
		}

		_, err = writer.WriteBody([]byte("Hello, World"))
		if err != nil {
			t.Fatal(err)
		}
	}

	if testutils.IsSingleTesting() {
		time.Sleep(2 * time.Second)
	}
}
