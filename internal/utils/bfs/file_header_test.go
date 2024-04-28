// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package bfs_test

import (
	"encoding/json"
	"github.com/TeaOSLab/EdgeNode/internal/utils/bfs"
	"github.com/TeaOSLab/EdgeNode/internal/utils/fasttime"
	"github.com/iwind/TeaGo/assert"
	"github.com/iwind/TeaGo/logs"
	"math/rand"
	"runtime"
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

func TestFileHeader_Compact_Merge(t *testing.T) {
	var a = assert.NewAssertion(t)

	var header = &bfs.FileHeader{
		Version: 1,
		Status:  200,
		HeaderBlocks: []bfs.BlockInfo{
			{
				BFileOffsetFrom:  1000,
				BFileOffsetTo:    1100,
				OriginOffsetFrom: 1200,
				OriginOffsetTo:   1300,
			},
			{
				BFileOffsetFrom:  1100,
				BFileOffsetTo:    1200,
				OriginOffsetFrom: 1300,
				OriginOffsetTo:   1400,
			},
		},
		BodyBlocks: []bfs.BlockInfo{
			{
				BFileOffsetFrom:  0,
				BFileOffsetTo:    100,
				OriginOffsetFrom: 200,
				OriginOffsetTo:   300,
			},
			{
				BFileOffsetFrom:  100,
				BFileOffsetTo:    200,
				OriginOffsetFrom: 300,
				OriginOffsetTo:   400,
			},
			{
				BFileOffsetFrom:  200,
				BFileOffsetTo:    300,
				OriginOffsetFrom: 400,
				OriginOffsetTo:   500,
			},
		},
	}
	header.Compact()
	logs.PrintAsJSON(header.HeaderBlocks)
	logs.PrintAsJSON(header.BodyBlocks)

	a.IsTrue(len(header.HeaderBlocks) == 1)
	a.IsTrue(len(header.BodyBlocks) == 1)
}

func TestFileHeader_Compact_Merge2(t *testing.T) {
	var header = &bfs.FileHeader{
		Version: 1,
		Status:  200,
		BodyBlocks: []bfs.BlockInfo{
			{
				BFileOffsetFrom:  0,
				BFileOffsetTo:    100,
				OriginOffsetFrom: 200,
				OriginOffsetTo:   300,
			},
			{
				BFileOffsetFrom:  101,
				BFileOffsetTo:    200,
				OriginOffsetFrom: 301,
				OriginOffsetTo:   400,
			},
			{
				BFileOffsetFrom:  200,
				BFileOffsetTo:    300,
				OriginOffsetFrom: 400,
				OriginOffsetTo:   500,
			},
		},
	}
	header.Compact()
	logs.PrintAsJSON(header.BodyBlocks)
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

func TestFileHeader_Encode(t *testing.T) {
	{
		var header = &bfs.FileHeader{
			Version:    1,
			Status:     200,
			ModifiedAt: fasttime.Now().Unix(),
			ExpiresAt:  fasttime.Now().Unix() + 3600,
			BodySize:   1 << 20,
			HeaderSize: 1 << 10,
			BodyBlocks: []bfs.BlockInfo{
				{
					BFileOffsetFrom: 1 << 10,
					BFileOffsetTo:   1 << 20,
				},
			},
		}
		data, err := header.Encode(bfs.Hash("123456"))
		if err != nil {
			t.Fatal(err)
		}
		jsonBytes, _ := json.Marshal(header)
		t.Log(len(header.BodyBlocks), "blocks", len(data), "bytes", "json:", len(jsonBytes), "bytes")

		_, _, _, err = bfs.DecodeMetaBlock(data)
		if err != nil {
			t.Fatal(err)
		}
	}

	{
		var header = &bfs.FileHeader{
			Version:    1,
			Status:     200,
			BodyBlocks: []bfs.BlockInfo{},
		}
		var offset int64
		for {
			var end = offset + 16<<10
			if end > 256<<10 {
				break
			}

			header.BodyBlocks = append(header.BodyBlocks, bfs.BlockInfo{
				BFileOffsetFrom: offset,
				BFileOffsetTo:   end,
			})

			offset = end
		}
		data, err := header.Encode(bfs.Hash("123456"))
		if err != nil {
			t.Fatal(err)
		}
		jsonBytes, _ := json.Marshal(header)
		t.Log(len(header.BodyBlocks), "blocks", len(data), "bytes", "json:", len(jsonBytes), "bytes")
	}

	{
		var header = &bfs.FileHeader{
			Version:    1,
			Status:     200,
			BodyBlocks: []bfs.BlockInfo{},
		}
		var offset int64
		for {
			var end = offset + 16<<10
			if end > 512<<10 {
				break
			}

			header.BodyBlocks = append(header.BodyBlocks, bfs.BlockInfo{
				BFileOffsetFrom: offset,
				BFileOffsetTo:   end,
			})

			offset = end
		}
		data, err := header.Encode(bfs.Hash("123456"))
		if err != nil {
			t.Fatal(err)
		}
		jsonBytes, _ := json.Marshal(header)
		t.Log(len(header.BodyBlocks), "blocks", len(data), "bytes", "json:", len(jsonBytes), "bytes")
	}

	{
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
		data, err := header.Encode(bfs.Hash("123456"))
		if err != nil {
			t.Fatal(err)
		}
		jsonBytes, _ := json.Marshal(header)
		t.Log(len(header.BodyBlocks), "blocks", len(data), "bytes", "json:", len(jsonBytes), "bytes")
	}
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

func BenchmarkFileHeader_Encode(b *testing.B) {
	runtime.GOMAXPROCS(12)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var header = &bfs.FileHeader{
				Version:    1,
				Status:     200,
				ModifiedAt: rand.Int63(),
				BodySize:   rand.Int63(),
				BodyBlocks: []bfs.BlockInfo{},
			}
			var offset int64
			for {
				var end = offset + 16<<10
				if end > 2<<20 {
					break
				}

				header.BodyBlocks = append(header.BodyBlocks, bfs.BlockInfo{
					BFileOffsetFrom: offset + int64(rand.Int()%1000000),
					BFileOffsetTo:   end + int64(rand.Int()%1000000),
				})

				offset = end
			}

			var hash = bfs.Hash("123456")

			_, err := header.Encode(hash)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
