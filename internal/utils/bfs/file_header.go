// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package bfs

import (
	"encoding/json"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"sort"
)

type FileHeader struct {
	Version         int         `json:"1,omitempty"`
	ModifiedAt      int64       `json:"2,omitempty"`
	ExpiresAt       int64       `json:"3,omitempty"`
	Status          int         `json:"4,omitempty"`
	HeaderSize      int64       `json:"5,omitempty"`
	BodySize        int64       `json:"6,omitempty"`
	ExpiredBodySize int64       `json:"7,omitempty"`
	HeaderBlocks    []BlockInfo `json:"8,omitempty"`
	BodyBlocks      []BlockInfo `json:"9,omitempty"`
	IsCompleted     bool        `json:"10,omitempty"`
	IsWriting       bool        `json:"11,omitempty"`
}

func (this *FileHeader) BlockAt(offset int64) (blockInfo BlockInfo, ok bool) {
	var l = len(this.BodyBlocks)
	if l == 1 {
		if this.BodyBlocks[0].Contains(offset) {
			return this.BodyBlocks[0], true
		}
		return
	}

	sort.Search(l, func(i int) bool {
		if this.BodyBlocks[i].Contains(offset) {
			blockInfo = this.BodyBlocks[i]
			ok = true
			return true
		}
		return this.BodyBlocks[i].OriginOffsetFrom > offset
	})

	return
}

func (this *FileHeader) MaxOffset() int64 {
	var l = len(this.BodyBlocks)
	if l > 0 {
		return this.BodyBlocks[l-1].OriginOffsetTo
	}
	return 0
}

func (this *FileHeader) Compact() {
	// TODO 合并相邻的headerBlocks和bodyBlocks（必须是对应的BFile offset也要相邻）

	if len(this.BodyBlocks) > 0 {
		sort.Slice(this.BodyBlocks, func(i, j int) bool {
			var block1 = this.BodyBlocks[i]
			var block2 = this.BodyBlocks[j]
			if block1.OriginOffsetFrom == block1.OriginOffsetFrom {
				return block1.OriginOffsetTo < block2.OriginOffsetTo
			}
			return block1.OriginOffsetFrom < block2.OriginOffsetFrom
		})

		var isCompleted = true
		if this.BodyBlocks[0].OriginOffsetFrom != 0 || this.BodyBlocks[len(this.BodyBlocks)-1].OriginOffsetTo != this.BodySize {
			isCompleted = false
		} else {
			for index, block := range this.BodyBlocks {
				// 是否有不连续的
				if index > 0 && block.OriginOffsetFrom > this.BodyBlocks[index-1].OriginOffsetTo {
					isCompleted = false
					break
				}
			}
		}
		this.IsCompleted = isCompleted
	}
}

func (this *FileHeader) Clone() *FileHeader {
	return &FileHeader{
		Version:         this.Version,
		ModifiedAt:      this.ModifiedAt,
		ExpiresAt:       this.ExpiresAt,
		Status:          this.Status,
		HeaderSize:      this.HeaderSize,
		BodySize:        this.BodySize,
		ExpiredBodySize: this.ExpiredBodySize,
		HeaderBlocks:    this.HeaderBlocks,
		BodyBlocks:      this.BodyBlocks,
		IsCompleted:     this.IsCompleted,
		IsWriting:       this.IsWriting,
	}
}

func (this *FileHeader) Encode(hash string) ([]byte, error) {
	headerJSON, err := json.Marshal(this)
	if err != nil {
		return nil, err
	}

	// we do not compress data which size is less than 100 bytes
	if len(headerJSON) < 100 {
		return EncodeMetaBlock(MetaActionNew, hash, append([]byte("json:"), headerJSON...))
	}

	var buf = utils.SharedBufferPool.Get()
	defer utils.SharedBufferPool.Put(buf)

	compressor, err := SharedCompressPool.Get(buf)
	if err != nil {
		return nil, err
	}

	_, err = compressor.Write(headerJSON)
	if err != nil {
		_ = compressor.Close()
		SharedCompressPool.Put(compressor)
		return nil, err
	}

	err = compressor.Close()
	SharedCompressPool.Put(compressor)
	if err != nil {
		return nil, err
	}

	return EncodeMetaBlock(MetaActionNew, hash, buf.Bytes())
}
