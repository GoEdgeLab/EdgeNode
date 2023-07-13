// Copyright 2023 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package syncutils

import (
	"sync"
)

type RWMutex struct {
	lockers      []*sync.RWMutex
	countLockers int
}

func NewRWMutex(count int) *RWMutex {
	if count <= 0 {
		count = 1
	}

	var lockers = []*sync.RWMutex{}
	for i := 0; i < count; i++ {
		lockers = append(lockers, &sync.RWMutex{})
	}

	return &RWMutex{
		lockers:      lockers,
		countLockers: len(lockers),
	}
}

func (this *RWMutex) Lock(index int) {
	this.lockers[index%this.countLockers].Lock()
}

func (this *RWMutex) Unlock(index int) {
	this.lockers[index%this.countLockers].Unlock()
}

func (this *RWMutex) RLock(index int) {
	this.lockers[index%this.countLockers].RLock()
}

func (this *RWMutex) RUnlock(index int) {
	this.lockers[index%this.countLockers].RUnlock()
}

func (this *RWMutex) TryLock(index int) bool {
	return this.lockers[index%this.countLockers].TryLock()
}

func (this *RWMutex) TryRLock(index int) bool {
	return this.lockers[index%this.countLockers].TryRLock()
}

func (this *RWMutex) RWMutex(index int) *sync.RWMutex {
	return this.lockers[index%this.countLockers]
}
