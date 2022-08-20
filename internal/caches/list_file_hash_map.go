// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package caches

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/TeaOSLab/EdgeNode/internal/zero"
	"sync"
)

// FileListHashMap 文件Hash列表
type FileListHashMap struct {
	m map[string]zero.Zero

	locker      sync.RWMutex
	isAvailable bool
	isReady     bool
}

func NewFileListHashMap() *FileListHashMap {
	return &FileListHashMap{
		m:           map[string]zero.Zero{},
		isAvailable: false,
		isReady:     false,
	}
}

func (this *FileListHashMap) Load(db *FileListDB) error {
	// 如果系统内存过小，我们不缓存
	if utils.SystemMemoryGB() < 3 {
		return nil
	}

	this.isAvailable = true

	var lastId int64
	for {
		hashList, maxId, err := db.ListHashes(lastId)
		if err != nil {
			return err
		}
		if len(hashList) == 0 {
			break
		}
		for _, hash := range hashList {
			this.Add(hash)
		}
		lastId = maxId
	}

	this.isReady = true
	return nil
}

func (this *FileListHashMap) Add(hash string) {
	if !this.isAvailable {
		return
	}

	this.locker.Lock()
	this.m[hash] = zero.New()
	this.locker.Unlock()
}

func (this *FileListHashMap) Delete(hash string) {
	if !this.isAvailable {
		return
	}

	this.locker.Lock()
	delete(this.m, hash)
	this.locker.Unlock()
}

func (this *FileListHashMap) Exist(hash string) bool {
	if !this.isAvailable {
		return true
	}
	if !this.isReady {
		// 只有完全Ready时才能判断是否为false
		return true
	}
	this.locker.RLock()
	_, ok := this.m[hash]
	this.locker.RUnlock()
	return ok
}

func (this *FileListHashMap) Clean() {
	this.locker.Lock()
	this.m = map[string]zero.Zero{}
	this.locker.Unlock()
}

func (this *FileListHashMap) IsReady() bool {
	return this.isReady
}
