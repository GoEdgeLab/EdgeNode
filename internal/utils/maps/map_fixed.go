// Copyright 2023 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package maputils

import "sync"

type KeyType interface {
	string | int | int64 | int32 | uint64 | uint32
}

type ValueType interface {
	any
}

// FixedMap
// TODO 解决已存在元素不能按顺序弹出的问题
type FixedMap[KeyT KeyType, ValueT ValueType] struct {
	m    map[KeyT]ValueT
	keys []KeyT

	maxSize int
	locker  sync.RWMutex
}

func NewFixedMap[KeyT KeyType, ValueT ValueType](maxSize int) *FixedMap[KeyT, ValueT] {
	return &FixedMap[KeyT, ValueT]{
		maxSize: maxSize,
		m:       map[KeyT]ValueT{},
	}
}

func (this *FixedMap[KeyT, ValueT]) Put(key KeyT, value ValueT) {
	this.locker.Lock()
	defer this.locker.Unlock()

	if this.maxSize <= 0 {
		return
	}

	_, exists := this.m[key]
	this.m[key] = value

	if !exists {
		this.keys = append(this.keys, key)

		if len(this.keys) > this.maxSize {
			var firstKey = this.keys[0]
			this.keys = this.keys[1:]
			delete(this.m, firstKey)
		}
	}
}

func (this *FixedMap[KeyT, ValueT]) Get(key KeyT) (value ValueT, ok bool) {
	this.locker.RLock()
	defer this.locker.RUnlock()
	value, ok = this.m[key]
	return
}

func (this *FixedMap[KeyT, ValueT]) Has(key KeyT) bool {
	this.locker.RLock()
	defer this.locker.RUnlock()
	_, ok := this.m[key]
	return ok
}

func (this *FixedMap[KeyT, ValueT]) Keys() []KeyT {
	this.locker.RLock()
	defer this.locker.RUnlock()
	return this.keys
}

func (this *FixedMap[KeyT, ValueT]) RawMap() map[KeyT]ValueT {
	this.locker.RLock()
	defer this.locker.RUnlock()
	return this.m
}
