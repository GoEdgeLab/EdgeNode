// Copyright 2023 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package syncutils

import (
	"runtime"
	"sync"
)

type KType interface {
	int | int16 | int32 | int64 | uint | uint16 | uint32 | uint64 | uintptr
}

type VType interface {
	any
}

type IntMap[K KType, V VType] struct {
	count   int
	m       []map[K]V
	lockers []*sync.RWMutex
}

func NewIntMap[K KType, V VType]() *IntMap[K, V] {
	var count = runtime.NumCPU() * 8
	if count <= 0 {
		count = 32
	}

	var m = []map[K]V{}
	var lockers = []*sync.RWMutex{}
	for i := 0; i < count; i++ {
		m = append(m, map[K]V{})
		lockers = append(lockers, &sync.RWMutex{})
	}

	return &IntMap[K, V]{
		count:   count,
		m:       m,
		lockers: lockers,
	}
}

func (this *IntMap[K, V]) Put(k K, v V) {
	var index = this.index(k)
	this.lockers[index].Lock()
	this.m[index][k] = v
	this.lockers[index].Unlock()
}

func (this *IntMap[K, V]) PutCompact(k K, v V, compactFunc func(oldV V, newV V) V) {
	var index = this.index(k)
	this.lockers[index].Lock()
	// 再次检查是否已经存在，如果已经存在则合并
	oldV, ok := this.m[index][k]
	if ok {
		this.m[index][k] = compactFunc(oldV, v)
	} else {
		this.m[index][k] = v
	}
	this.lockers[index].Unlock()
}

func (this *IntMap[K, V]) Has(k K) bool {
	var index = this.index(k)
	this.lockers[index].RLock()
	_, ok := this.m[index][k]
	this.lockers[index].RUnlock()
	return ok
}

func (this *IntMap[K, V]) Get(k K) (value V) {
	var index = this.index(k)
	this.lockers[index].RLock()
	value = this.m[index][k]
	this.lockers[index].RUnlock()
	return
}

func (this *IntMap[K, V]) GetOk(k K) (value V, ok bool) {
	var index = this.index(k)
	this.lockers[index].RLock()
	value, ok = this.m[index][k]
	this.lockers[index].RUnlock()
	return
}

func (this *IntMap[K, V]) Delete(k K) {
	var index = this.index(k)
	this.lockers[index].Lock()
	delete(this.m[index], k)
	this.lockers[index].Unlock()
}

func (this *IntMap[K, V]) DeleteUnsafe(k K) {
	var index = this.index(k)
	delete(this.m[index], k)
}

func (this *IntMap[K, V]) Len() int {
	var l int
	for i := 0; i < this.count; i++ {
		this.lockers[i].RLock()
		l += len(this.m[i])
		this.lockers[i].RUnlock()
	}
	return l
}

func (this *IntMap[K, V]) ForEachRead(iterator func(k K, v V)) {
	for i := 0; i < this.count; i++ {
		this.lockers[i].RLock()
		for k, v := range this.m[i] {
			iterator(k, v)
		}
		this.lockers[i].RUnlock()
	}
}

func (this *IntMap[K, V]) ForEachWrite(iterator func(k K, v V)) {
	for i := 0; i < this.count; i++ {
		this.lockers[i].Lock()
		for k, v := range this.m[i] {
			iterator(k, v)
		}
		this.lockers[i].Unlock()
	}
}

func (this *IntMap[K, V]) index(k K) int {
	var index = int(k % K(this.count))
	if index < 0 {
		index = -index
	}
	return index
}
