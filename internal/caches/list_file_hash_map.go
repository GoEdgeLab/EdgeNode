// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package caches

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/TeaOSLab/EdgeNode/internal/zero"
	"math/big"
	"sync"
)

const HashMapSharding = 31

var bigIntPool = sync.Pool{
	New: func() any {
		return big.NewInt(0)
	},
}

// FileListHashMap 文件Hash列表
type FileListHashMap struct {
	m []map[uint64]zero.Zero

	lockers []*sync.RWMutex

	isAvailable bool
	isReady     bool
}

func NewFileListHashMap() *FileListHashMap {
	var m = make([]map[uint64]zero.Zero, HashMapSharding)
	var lockers = make([]*sync.RWMutex, HashMapSharding)

	for i := 0; i < HashMapSharding; i++ {
		m[i] = map[uint64]zero.Zero{}
		lockers[i] = &sync.RWMutex{}
	}

	return &FileListHashMap{
		m:           m,
		lockers:     lockers,
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
	var maxLoops = 50_000
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

		maxLoops--
		if maxLoops <= 0 {
			break
		}
	}

	this.isReady = true
	return nil
}

func (this *FileListHashMap) Add(hash string) {
	if !this.isAvailable {
		return
	}

	hashInt, index := this.bigInt(hash)

	this.lockers[index].Lock()
	this.m[index][hashInt] = zero.New()
	this.lockers[index].Unlock()
}

func (this *FileListHashMap) AddHashes(hashes []string) {
	if !this.isAvailable {
		return
	}

	for _, hash := range hashes {
		hashInt, index := this.bigInt(hash)
		this.lockers[index].Lock()
		this.m[index][hashInt] = zero.New()
		this.lockers[index].Unlock()
	}
}

func (this *FileListHashMap) Delete(hash string) {
	if !this.isAvailable {
		return
	}

	hashInt, index := this.bigInt(hash)
	this.lockers[index].Lock()
	delete(this.m[index], hashInt)
	this.lockers[index].Unlock()
}

func (this *FileListHashMap) Exist(hash string) bool {
	if !this.isAvailable {
		return true
	}
	if !this.isReady {
		// 只有完全Ready时才能判断是否为false
		return true
	}

	hashInt, index := this.bigInt(hash)

	this.lockers[index].RLock()
	_, ok := this.m[index][hashInt]
	this.lockers[index].RUnlock()
	return ok
}

func (this *FileListHashMap) Clean() {
	for i := 0; i < HashMapSharding; i++ {
		this.lockers[i].Lock()
	}

	this.m = make([]map[uint64]zero.Zero, HashMapSharding)

	for i := HashMapSharding - 1; i >= 0; i-- {
		this.lockers[i].Unlock()
	}
}

func (this *FileListHashMap) IsReady() bool {
	return this.isReady
}

func (this *FileListHashMap) Len() int {
	for i := 0; i < HashMapSharding; i++ {
		this.lockers[i].Lock()
	}

	var count = 0
	for _, shard := range this.m {
		count += len(shard)
	}

	for i := HashMapSharding - 1; i >= 0; i-- {
		this.lockers[i].Unlock()
	}

	return count
}

func (this *FileListHashMap) SetIsAvailable(isAvailable bool) {
	this.isAvailable = isAvailable
}

func (this *FileListHashMap) SetIsReady(isReady bool) {
	this.isReady = isReady
}

func (this *FileListHashMap) bigInt(hash string) (hashInt uint64, index int) {
	var bigInt = bigIntPool.Get().(*big.Int)
	bigInt.SetString(hash, 16)
	hashInt = bigInt.Uint64()
	bigIntPool.Put(bigInt)

	index = int(hashInt % HashMapSharding)
	return
}
