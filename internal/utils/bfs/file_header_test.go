// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package bfs_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils/bfs"
	"github.com/iwind/TeaGo/assert"
	"github.com/iwind/TeaGo/logs"
	"testing"
)

func TestFileHeader_Compact(t *testing.T) {
	var a = assert.NewAssertion(t)

	{
		var header = &bfs.FileHeader{
			Version:  1,
			Status:   200,
			BodySize: 100,
			BodyBlocks: []bfs.BlockInfo{
				{
					OriginOffsetFrom: 0,
					OriginOffsetTo:   100,
				},
			},
		}
		header.Compact()
		a.IsTrue(header.IsCompleted)
	}

	{
		var header = &bfs.FileHeader{
			Version:  1,
			Status:   200,
			BodySize: 200,
			BodyBlocks: []bfs.BlockInfo{
				{
					OriginOffsetFrom: 100,
					OriginOffsetTo:   200,
				},
				{
					OriginOffsetFrom: 0,
					OriginOffsetTo:   100,
				},
			},
		}
		header.Compact()
		a.IsTrue(header.IsCompleted)
	}

	{
		var header = &bfs.FileHeader{
			Version:  1,
			Status:   200,
			BodySize: 200,
			BodyBlocks: []bfs.BlockInfo{
				{
					OriginOffsetFrom: 10,
					OriginOffsetTo:   99,
				},
				{
					OriginOffsetFrom: 110,
					OriginOffsetTo:   200,
				},
				{
					OriginOffsetFrom: 88,
					OriginOffsetTo:   120,
				},
				{
					OriginOffsetFrom: 0,
					OriginOffsetTo:   100,
				},
			},
		}
		header.Compact()
		a.IsTrue(header.IsCompleted)
	}

	{
		var header = &bfs.FileHeader{
			Version:  1,
			Status:   200,
			BodySize: 100,
			BodyBlocks: []bfs.BlockInfo{
				{
					OriginOffsetFrom: 10,
					OriginOffsetTo:   100,
				},
				{
					OriginOffsetFrom: 100,
					OriginOffsetTo:   200,
				},
			},
		}
		header.Compact()
		a.IsFalse(header.IsCompleted)
	}

	{
		var header = &bfs.FileHeader{
			Version:  1,
			Status:   200,
			BodySize: 200,
			BodyBlocks: []bfs.BlockInfo{
				{
					OriginOffsetFrom: 0,
					OriginOffsetTo:   100,
				},
				{
					OriginOffsetFrom: 100,
					OriginOffsetTo:   199,
				},
			},
		}
		header.Compact()
		a.IsFalse(header.IsCompleted)
	}

	{
		var header = &bfs.FileHeader{
			Version:  1,
			Status:   200,
			BodySize: 200,
			BodyBlocks: []bfs.BlockInfo{
				{
					OriginOffsetFrom: 0,
					OriginOffsetTo:   100,
				},
				{
					OriginOffsetFrom: 101,
					OriginOffsetTo:   200,
				},
			},
		}
		header.Compact()
		a.IsFalse(header.IsCompleted)
	}
}

func TestFileHeader_Clone(t *testing.T) {
	var a = assert.NewAssertion(t)

	var header = &bfs.FileHeader{
		Version: 1,
		Status:  200,
		BodyBlocks: []bfs.BlockInfo{
			{
				BFileOffsetFrom: 0,
				BFileOffsetTo:   100,
			},
		},
	}

	var clonedHeader = header.Clone()
	t.Log("=== cloned header ===")
	logs.PrintAsJSON(clonedHeader, t)
	a.IsTrue(len(clonedHeader.BodyBlocks) == 1)

	header.BodyBlocks = append(header.BodyBlocks, bfs.BlockInfo{
		BFileOffsetFrom: 100,
		BFileOffsetTo:   200,
	})
	header.BodyBlocks = append(header.BodyBlocks, bfs.BlockInfo{
		BFileOffsetFrom: 300,
		BFileOffsetTo:   400,
	})

	clonedHeader.BodyBlocks[0].OriginOffsetFrom = 100000000

	t.Log("=== after changed ===")
	logs.PrintAsJSON(clonedHeader, t)
	a.IsTrue(len(clonedHeader.BodyBlocks) == 1)

	t.Log("=== original header ===")
	logs.PrintAsJSON(header, t)
	a.IsTrue(header.BodyBlocks[0].OriginOffsetFrom != clonedHeader.BodyBlocks[0].OriginOffsetFrom)
}

func BenchmarkFileHeader_Compact(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var header = &bfs.FileHeader{
			Version:    1,
			Status:     200,
			BodySize:   200,
			BodyBlocks: nil,
		}

		for j := 0; j < 100; j++ {
			header.BodyBlocks = append(header.BodyBlocks, bfs.BlockInfo{
				OriginOffsetFrom: int64(j * 100),
				OriginOffsetTo:   int64(j * 200),
				BFileOffsetFrom:  0,
				BFileOffsetTo:    0,
			})
		}

		header.Compact()
	}
}
