// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package caches

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils/fasttime"
	"github.com/TeaOSLab/EdgeNode/internal/utils/linkedlist"
)

type OpenFilePool struct {
	c        chan *OpenFile
	linkItem *linkedlist.Item[*OpenFilePool]
	filename string
	version  int64
	isClosed bool
	usedSize int64
}

func NewOpenFilePool(filename string) *OpenFilePool {
	var pool = &OpenFilePool{
		filename: filename,
		c:        make(chan *OpenFile, 1024),
		version:  fasttime.Now().UnixMilli(),
	}
	pool.linkItem = linkedlist.NewItem[*OpenFilePool](pool)
	return pool
}

func (this *OpenFilePool) Filename() string {
	return this.filename
}

func (this *OpenFilePool) Get() (resultFile *OpenFile, consumed bool, consumedSize int64) {
	// 如果已经关闭，直接返回
	if this.isClosed {
		return nil, false, 0
	}

	select {
	case file := <-this.c:
		if file != nil {
			this.usedSize -= file.size

			err := file.SeekStart()
			if err != nil {
				_ = file.Close()
				return nil, true, file.size
			}
			file.version = this.version

			return file, true, file.size
		}
		return nil, false, 0
	default:
		return nil, false, 0
	}
}

func (this *OpenFilePool) Put(file *OpenFile) bool {
	// 如果已关闭，则不接受新的文件
	if this.isClosed {
		_ = file.Close()
		return false
	}

	// 检查文件版本号
	if this.version > 0 && file.version > 0 && file.version != this.version {
		_ = file.Close()
		return false
	}

	// 加入Pool
	select {
	case this.c <- file:
		this.usedSize += file.size
		return true
	default:
		// 多余的直接关闭
		_ = file.Close()
		return false
	}
}

func (this *OpenFilePool) Len() int {
	return len(this.c)
}

func (this *OpenFilePool) TotalSize() int64 {
	return this.usedSize
}

func (this *OpenFilePool) SetClosing() {
	this.isClosed = true
}

func (this *OpenFilePool) Close() {
	this.isClosed = true
	for {
		select {
		case file := <-this.c:
			_ = file.Close()
		default:
			return
		}
	}
}
