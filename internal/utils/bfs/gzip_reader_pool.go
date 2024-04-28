// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package bfs

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils/percpu"
	"github.com/klauspost/compress/gzip"
	"io"
	"runtime"
)

var SharedDecompressPool = NewGzipReaderPool()

type GzipReaderPool struct {
	c     chan *gzip.Reader
	cList []chan *gzip.Reader
}

func NewGzipReaderPool() *GzipReaderPool {
	const poolSize = 16

	var countProcs = runtime.GOMAXPROCS(0)
	if countProcs <= 0 {
		countProcs = runtime.NumCPU()
	}
	countProcs *= 4

	var cList []chan *gzip.Reader
	for i := 0; i < countProcs; i++ {
		cList = append(cList, make(chan *gzip.Reader, poolSize))
	}

	return &GzipReaderPool{
		c:     make(chan *gzip.Reader, poolSize),
		cList: cList,
	}
}

func (this *GzipReaderPool) Get(rawReader io.Reader) (*gzip.Reader, error) {
	select {
	case w := <-this.getC():
		err := w.Reset(rawReader)
		if err != nil {
			return nil, err
		}
		return w, nil
	default:
		return gzip.NewReader(rawReader)
	}
}

func (this *GzipReaderPool) Put(reader *gzip.Reader) {
	select {
	case this.getC() <- reader:
	default:
		// 不需要close，因为已经在使用的时候调用了
	}
}

func (this *GzipReaderPool) getC() chan *gzip.Reader {
	var procId = percpu.GetProcId()
	if procId < len(this.cList) {
		return this.cList[procId]
	}
	return this.c
}
