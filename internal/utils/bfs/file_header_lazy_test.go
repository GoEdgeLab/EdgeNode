// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package bfs_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils/bfs"
	"runtime"
	"testing"
)

func TestNewLazyFileHeaderFromData(t *testing.T) {
	var header = &bfs.FileHeader{
		Version: 1,
		Status:  200,
		BodyBlocks: []bfs.BlockInfo{
			{
				BFileOffsetFrom: 0,
				BFileOffsetTo:   1 << 20,
			},
		},
	}
	blockBytes, err := header.Encode(bfs.Hash("123456"))
	if err != nil {
		t.Fatal(err)
	}

	_, _, rawData, err := bfs.DecodeMetaBlock(blockBytes)
	if err != nil {
		t.Fatal(err)
	}

	var lazyHeader = bfs.NewLazyFileHeaderFromData(rawData)
	newHeader, err := lazyHeader.FileHeaderUnsafe()
	if err != nil {
		t.Fatal(err)
	}

	t.Log(newHeader)
}

func BenchmarkLazyFileHeader_Decode(b *testing.B) {
	runtime.GOMAXPROCS(12)

	var header = &bfs.FileHeader{
		Version:    1,
		Status:     200,
		BodyBlocks: []bfs.BlockInfo{},
	}
	var offset int64
	for {
		var end = offset + 16<<10
		if end > 1<<20 {
			break
		}

		header.BodyBlocks = append(header.BodyBlocks, bfs.BlockInfo{
			BFileOffsetFrom: offset,
			BFileOffsetTo:   end,
		})

		offset = end
	}

	var hash = bfs.Hash("123456")

	blockBytes, err := header.Encode(hash)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _, rawData, decodeErr := bfs.DecodeMetaBlock(blockBytes)
			if decodeErr != nil {
				b.Fatal(decodeErr)
			}

			var lazyHeader = bfs.NewLazyFileHeaderFromData(rawData)
			_, decodeErr = lazyHeader.FileHeaderUnsafe()
			if decodeErr != nil {
				b.Fatal(decodeErr)
			}
		}
	})
}
