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

// SQLiteFileListHashMap 文件Hash列表
type SQLiteFileListHashMap struct {
	m []map[uint64]zero.Zero

	lockers []*sync.RWMutex

	isAvailable bool
	isReady     bool
}

func NewSQLiteFileListHashMap() *SQLiteFileListHashMap {
	var m = make([]map[uint64]zero.Zero, HashMapSharding)
	var lockers = make([]*sync.RWMutex, HashMapSharding)

	for i := 0; i < HashMapSharding; i++ {
		m[i] = map[uint64]zero.Zero{}
		lockers[i] = &sync.RWMutex{}
	}

	return &SQLiteFileListHashMap{
		m:           m,
		lockers:     lockers,
		isAvailable: false,
		isReady:     false,
	}
}

func (this *SQLiteFileListHashMap) Load(db *SQLiteFileListDB) error {
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

func (this *SQLiteFileListHashMap) Add(hash string) {
	if !this.isAvailable {
		return
	}

	hashInt, index := this.bigInt(hash)

	this.lockers[index].Lock()
	this.m[index][hashInt] = zero.New()
	this.lockers[index].Unlock()
}

func (this *SQLiteFileListHashMap) AddHashes(hashes []string) {
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

func (this *SQLiteFileListHashMap) Delete(hash string) {
	if !this.isAvailable {
		return
	}

	hashInt, index := this.bigInt(hash)
	this.lockers[index].Lock()
	delete(this.m[index], hashInt)
	this.lockers[index].Unlock()
}

func (this *SQLiteFileListHashMap) Exist(hash string) bool {
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

func (this *SQLiteFileListHashMap) Clean() {
	for i := 0; i < HashMapSharding; i++ {
		this.lockers[i].Lock()
	}

	// 这里不能简单清空 this.m ，避免导致别的数据无法写入 map 而产生 panic
	for i := 0; i < HashMapSharding; i++ {
		this.m[i] = map[uint64]zero.Zero{}
	}

	for i := HashMapSharding - 1; i >= 0; i-- {
		this.lockers[i].Unlock()
	}
}

func (this *SQLiteFileListHashMap) IsReady() bool {
	return this.isReady
}

func (this *SQLiteFileListHashMap) Len() int {
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

func (this *SQLiteFileListHashMap) SetIsAvailable(isAvailable bool) {
	this.isAvailable = isAvailable
}

func (this *SQLiteFileListHashMap) SetIsReady(isReady bool) {
	this.isReady = isReady
}

func (this *SQLiteFileListHashMap) bigInt(hash string) (hashInt uint64, index int) {
	var bigInt = bigIntPool.Get().(*big.Int)
	bigInt.SetString(hash, 16)
	hashInt = bigInt.Uint64()
	bigIntPool.Put(bigInt)

	index = int(hashInt % HashMapSharding)
	return
}
