// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package bfs

type BlockFileOptions struct {
	BytesPerSync int64
	MaxOpenFiles int // TODO 需要实现，主要用于OpenFileReader
}

func (this *BlockFileOptions) EnsureDefaults() {
	if this.BytesPerSync <= 0 {
		this.BytesPerSync = 1 << 20
	}
}

var DefaultBlockFileOptions = &BlockFileOptions{
	BytesPerSync: 1 << 20,
}
