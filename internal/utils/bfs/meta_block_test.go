// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package bfs_test

import (
	"bytes"
	"github.com/TeaOSLab/EdgeNode/internal/utils/bfs"
	"github.com/iwind/TeaGo/assert"
	"testing"
)

func TestMetaBlock(t *testing.T) {
	var a = assert.NewAssertion(t)

	{
		var srcHash = bfs.Hash("a")
		b, err := bfs.EncodeMetaBlock(bfs.MetaActionNew, srcHash, []byte{1, 2, 3})
		if err != nil {
			t.Fatal(err)
		}
		t.Log(b)

		{
			action, hash, data, decodeErr := bfs.DecodeMetaBlock(b)
			if decodeErr != nil {
				t.Fatal(err)
			}
			a.IsTrue(action == bfs.MetaActionNew)
			a.IsTrue(hash == srcHash)
			a.IsTrue(bytes.Equal(data, []byte{1, 2, 3}))
		}
	}

	{
		var srcHash = bfs.Hash("bcd")

		b, err := bfs.EncodeMetaBlock(bfs.MetaActionRemove, srcHash, []byte{1, 2, 3})
		if err != nil {
			t.Fatal(err)
		}
		t.Log(b)
		{
			action, hash, data, decodeErr := bfs.DecodeMetaBlock(b)
			if decodeErr != nil {
				t.Fatal(err)
			}
			a.IsTrue(action == bfs.MetaActionRemove)
			a.IsTrue(hash == srcHash)
			a.IsTrue(len(data) == 0)
		}
	}
}
