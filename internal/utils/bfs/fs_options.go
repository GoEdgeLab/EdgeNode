// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package bfs

import "time"

type FSOptions struct {
	MaxOpenFiles int
	BytesPerSync int64
	SyncTimeout  time.Duration
	MaxSyncFiles int
}

func (this *FSOptions) EnsureDefaults() {
	if this.MaxOpenFiles <= 0 {
		this.MaxOpenFiles = 4 << 10
	}
	if this.BytesPerSync <= 0 {
		this.BytesPerSync = 1 << 20 // TODO 根据硬盘实际写入速度进行调整
	}
	if this.SyncTimeout <= 0 {
		this.SyncTimeout = 1 * time.Second
	}
	if this.MaxSyncFiles <= 0 {
		this.MaxSyncFiles = 32
	}
}

var DefaultFSOptions = &FSOptions{
	MaxOpenFiles: 4 << 10,
	BytesPerSync: 1 << 20, // TODO 根据硬盘实际写入速度进行调整
	SyncTimeout:  1 * time.Second,
	MaxSyncFiles: 32,
}
