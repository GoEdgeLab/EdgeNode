// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package readers

import (
	"errors"
	"github.com/iwind/TeaGo/types"
	"io"
	"sync"
)

type concurrentSubReader struct {
	main  *ConcurrentReaderList
	index int
}

func (this *concurrentSubReader) Read(p []byte) (n int, err error) {
	n, err = this.main.readIndex(p, this.index)
	this.index++
	return
}

func (this *concurrentSubReader) Close() error {
	this.main.removeSubReader(this)

	err := this.main.Close()
	if err != nil {
		return err
	}
	return nil
}

// ConcurrentReaderList
// TODO 动态调整 pieces = pieces[minPieceIndex:] 以节约内存
type ConcurrentReaderList struct {
	locker     sync.RWMutex
	readLocker sync.Mutex

	mainReader   io.ReadCloser
	subReaderMap map[*concurrentSubReader]bool
	pieces       [][]byte
	lastErr      error
}

func NewConcurrentReaderList(mainReader io.ReadCloser) *ConcurrentReaderList {
	return &ConcurrentReaderList{
		mainReader:   mainReader,
		subReaderMap: map[*concurrentSubReader]bool{},
	}
}

func (this *ConcurrentReaderList) NewReader() io.ReadCloser {
	var subReader = &concurrentSubReader{
		main: this,
	}
	this.locker.Lock()
	this.subReaderMap[subReader] = true
	this.locker.Unlock()
	return subReader
}

func (this *ConcurrentReaderList) read(p []byte) (n int, err error) {
	n, err = this.mainReader.Read(p)
	this.lastErr = err

	if n > 0 {
		var piece = make([]byte, n)
		copy(piece, p[:n])
		this.locker.Lock()
		this.pieces = append(this.pieces, piece)
		this.locker.Unlock()
	}

	return
}

func (this *ConcurrentReaderList) readIndex(p []byte, index int) (n int, err error) {
	// 如果已经有数据
	this.locker.RLock()
	var countPieces = len(this.pieces)
	if index < countPieces {
		var piece = this.pieces[index]
		this.locker.RUnlock()
		var pn = len(piece)
		if len(p) < pn {
			err = errors.New("invalid buffer length '" + types.String(len(p)) + "' vs '" + types.String(len(piece)) + "'")
			return
		}
		n = pn
		copy(p, piece)
		return
	}
	this.locker.RUnlock()

	if this.lastErr != nil {
		return 0, this.lastErr
	}

	// 如果没有数据，则读取之
	this.readLocker.Lock()

	// 再次检查数据是否已更新
	this.locker.RLock()
	if len(this.pieces) > countPieces || this.lastErr != nil {
		this.locker.RUnlock()
		this.readLocker.Unlock()
		return this.readIndex(p, index)
	}
	this.locker.RUnlock()

	// 从原始Reader中读取
	n, err = this.read(p)
	this.readLocker.Unlock()
	if n > 0 {
		// 重新尝试
		return this.readIndex(p, index)
	}
	return
}

func (this *ConcurrentReaderList) removeSubReader(subReader *concurrentSubReader) {
	this.locker.Lock()
	delete(this.subReaderMap, subReader)
	this.locker.Unlock()
}

func (this *ConcurrentReaderList) Close() error {
	this.locker.Lock()
	if len(this.subReaderMap) == 0 {
		this.locker.Unlock()
		return this.mainReader.Close()
	}
	this.locker.Unlock()
	return nil
}
