// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package caches

import (
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/utils/fnv"
	fsutils "github.com/TeaOSLab/EdgeNode/internal/utils/fs"
	memutils "github.com/TeaOSLab/EdgeNode/internal/utils/mem"
	"sync"
)

var SharedPartialRangesQueue = NewPartialRangesQueue()

func init() {
	if !teaconst.IsMain {
		return
	}

	SharedPartialRangesQueue.Start()
}

const partialRangesQueueSharding = 8

// PartialRangesQueue ranges file writing queue
type PartialRangesQueue struct {
	m [partialRangesQueueSharding]map[string][]byte // { filename => data, ... }

	c  chan string // filename1, ...
	mu [partialRangesQueueSharding]*sync.RWMutex
}

// NewPartialRangesQueue Create new queue
func NewPartialRangesQueue() *PartialRangesQueue {
	var queueSize = 512
	var memGB = memutils.SystemMemoryGB()
	if memGB > 16 {
		queueSize = 8 << 10
	} else if memGB > 8 {
		queueSize = 4 << 10
	} else if memGB > 4 {
		queueSize = 2 << 10
	} else if memGB > 2 {
		queueSize = 1 << 10
	}

	var m = [partialRangesQueueSharding]map[string][]byte{}
	var muList = [partialRangesQueueSharding]*sync.RWMutex{}
	for i := 0; i < partialRangesQueueSharding; i++ {
		muList[i] = &sync.RWMutex{}
		m[i] = map[string][]byte{}
	}

	return &PartialRangesQueue{
		mu: muList,
		m:  m,
		c:  make(chan string, queueSize),
	}
}

// Start the queue
func (this *PartialRangesQueue) Start() {
	goman.New(func() {
		this.Dump()
	})
}

// Put ranges data to filename
func (this *PartialRangesQueue) Put(filename string, data []byte) {
	var index = this.indexForKey(filename)

	this.mu[index].Lock()
	this.m[index][filename] = data
	this.mu[index].Unlock()

	// always wait to finish
	this.c <- filename
}

// Get ranges data from filename
func (this *PartialRangesQueue) Get(filename string) ([]byte, error) {
	var index = this.indexForKey(filename)

	this.mu[index].RLock()
	data, ok := this.m[index][filename]
	this.mu[index].RUnlock()

	if ok {
		return data, nil
	}

	return fsutils.ReadFile(filename)
}

// Delete ranges filename
func (this *PartialRangesQueue) Delete(filename string) {
	var index = this.indexForKey(filename)

	this.mu[index].Lock()
	delete(this.m[index], filename)
	this.mu[index].Unlock()
}

// Dump ranges to filename from memory
func (this *PartialRangesQueue) Dump() {
	for filename := range this.c {
		var index = this.indexForKey(filename)

		this.mu[index].Lock()
		data, ok := this.m[index][filename]
		if ok {
			delete(this.m[index], filename)
		}
		this.mu[index].Unlock()

		if !ok || len(data) == 0 {
			continue
		}

		err := fsutils.WriteFile(filename, data, 0666)
		if err != nil {
			remotelogs.Println("PARTIAL_RANGES_QUEUE", "write file '"+filename+"' failed: "+err.Error())
		}
	}
}

// Len count all files
func (this *PartialRangesQueue) Len() int {
	var count int

	for i := 0; i < partialRangesQueueSharding; i++ {
		this.mu[i].RLock()
		count += len(this.m[i])
		this.mu[i].RUnlock()
	}

	return count
}

func (this *PartialRangesQueue) indexForKey(filename string) int {
	return int(fnv.HashString(filename) % partialRangesQueueSharding)
}
