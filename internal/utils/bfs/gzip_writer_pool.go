// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package bfs

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils/percpu"
	"github.com/klauspost/compress/gzip"
	"io"
	"runtime"
)

var SharedCompressPool = NewGzipWriterPool()

type GzipWriterPool struct {
	c     chan *gzip.Writer
	cList []chan *gzip.Writer
}

func NewGzipWriterPool() *GzipWriterPool {
	const poolSize = 16

	var countProcs = runtime.GOMAXPROCS(0)
	if countProcs <= 0 {
		countProcs = runtime.NumCPU()
	}
	countProcs *= 4

	var cList []chan *gzip.Writer
	for i := 0; i < countProcs; i++ {
		cList = append(cList, make(chan *gzip.Writer, poolSize))
	}

	return &GzipWriterPool{
		c:     make(chan *gzip.Writer, poolSize),
		cList: cList,
	}
}

func (this *GzipWriterPool) Get(rawWriter io.Writer) (*gzip.Writer, error) {
	select {
	case w := <-this.getC():
		w.Reset(rawWriter)
		return w, nil
	default:
		return gzip.NewWriterLevel(rawWriter, gzip.BestSpeed)
	}
}

func (this *GzipWriterPool) Put(writer *gzip.Writer) {
	select {
	case this.getC() <- writer:
	default:
		// 不需要close，因为已经在使用的时候调用了
	}
}

func (this *GzipWriterPool) getC() chan *gzip.Writer {
	var procId = percpu.GetProcId()
	if procId < len(this.cList) {
		return this.cList[procId]
	}
	return this.c
}
