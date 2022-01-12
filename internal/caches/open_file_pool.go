// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package caches

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/TeaOSLab/EdgeNode/internal/utils/linkedlist"
)

type OpenFilePool struct {
	c        chan *OpenFile
	linkItem *linkedlist.Item
	filename string
	version  int64
}

func NewOpenFilePool(filename string) *OpenFilePool {
	var pool = &OpenFilePool{
		filename: filename,
		c:        make(chan *OpenFile, 1024),
		version:  utils.UnixTimeMilli(),
	}
	pool.linkItem = linkedlist.NewItem(pool)
	return pool
}

func (this *OpenFilePool) Filename() string {
	return this.filename
}

func (this *OpenFilePool) Get() (*OpenFile, bool) {
	select {
	case file := <-this.c:
		err := file.SeekStart()
		if err != nil {
			_ = file.Close()
			return nil, true
		}
		file.version = this.version

		return file, true
	default:
		return nil, false
	}
}

func (this *OpenFilePool) Put(file *OpenFile) bool {
	if file.version > 0 && file.version != this.version {
		_ = file.Close()
		return false
	}
	select {
	case this.c <- file:
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

func (this *OpenFilePool) Close() {
Loop:
	for {
		select {
		case file := <-this.c:
			_ = file.Close()
		default:
			break Loop
		}
	}
}
