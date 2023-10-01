// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package caches

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/TeaOSLab/EdgeNode/internal/zero"
	"math/big"
	"sync"
)

// FileListHashMap 文件Hash列表
type FileListHashMap struct {
	m map[uint64]zero.Zero

	locker      sync.RWMutex
	isAvailable bool
	isReady     bool
}

func NewFileListHashMap() *FileListHashMap {
	return &FileListHashMap{
		m:           map[uint64]zero.Zero{},
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
		this.AddHashes(hashList)
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
	this.m[this.bigInt(hash)] = zero.New()
	this.locker.Unlock()
}

func (this *FileListHashMap) AddHashes(hashes []string) {
	if !this.isAvailable {
		return
	}

	this.locker.Lock()
	for _, hash := range hashes {
		this.m[this.bigInt(hash)] = zero.New()
	}
	this.locker.Unlock()
}

func (this *FileListHashMap) Delete(hash string) {
	if !this.isAvailable {
		return
	}

	this.locker.Lock()
	delete(this.m, this.bigInt(hash))
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
	_, ok := this.m[this.bigInt(hash)]
	this.locker.RUnlock()
	return ok
}

func (this *FileListHashMap) Clean() {
	this.locker.Lock()
	this.m = map[uint64]zero.Zero{}
	this.locker.Unlock()
}

func (this *FileListHashMap) IsReady() bool {
	return this.isReady
}

func (this *FileListHashMap) Len() int {
	this.locker.Lock()
	defer this.locker.Unlock()
	return len(this.m)
}

func (this *FileListHashMap) SetIsAvailable(isAvailable bool) {
	this.isAvailable = isAvailable
}

func (this *FileListHashMap) bigInt(hash string) uint64 {
	var bigInt = big.NewInt(0)
	bigInt.SetString(hash, 16)
	return bigInt.Uint64()
}
