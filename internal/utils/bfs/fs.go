// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package bfs

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type FS struct {
	dir string
	opt *FSOptions

	bMap     map[string]*BlocksFile // name => *BlocksFile
	mu       *sync.RWMutex
	isClosed bool

	syncTicker *time.Ticker
}

func NewFS(dir string, options *FSOptions) *FS {
	options.EnsureDefaults()

	var fs = &FS{
		dir:        dir,
		bMap:       map[string]*BlocksFile{},
		mu:         &sync.RWMutex{},
		opt:        options,
		syncTicker: time.NewTicker(1 * time.Second),
	}
	go fs.init()
	return fs
}

func (this *FS) init() {
	// sync in background
	for range this.syncTicker.C {
		this.syncLoop()
	}
}

func (this *FS) OpenFileWriter(hash string, bodySize int64, isPartial bool) (*FileWriter, error) {
	err := CheckHashErr(hash)
	if err != nil {
		return nil, err
	}

	if isPartial && bodySize <= 0 {
		return nil, errors.New("invalid body size for partial content")
	}

	bPath, bName, err := this.bPathForHash(hash)
	if err != nil {
		return nil, err
	}

	// check directory
	// TODO 需要改成提示找不到文件的时候再检查
	_, err = os.Stat(filepath.Dir(bPath))
	if err != nil && os.IsNotExist(err) {
		_ = os.MkdirAll(filepath.Dir(bPath), 0777)
	}

	this.mu.Lock()
	defer this.mu.Unlock()
	bFile, ok := this.bMap[bName]
	if ok {
		return bFile.OpenFileWriter(hash, bodySize, isPartial)
	}

	bFile, err = NewBlocksFile(bPath, &BlockFileOptions{
		BytesPerSync: this.opt.BytesPerSync,
	})
	if err != nil {
		return nil, err
	}
	this.bMap[bName] = bFile
	return bFile.OpenFileWriter(hash, bodySize, isPartial)
}

func (this *FS) OpenFileReader(hash string, isPartial bool) (*FileReader, error) {
	err := CheckHashErr(hash)
	if err != nil {
		return nil, err
	}

	_, bName, err := this.bPathForHash(hash)
	if err != nil {
		return nil, err
	}

	this.mu.Lock()
	defer this.mu.Unlock()
	bFile, ok := this.bMap[bName]
	if ok {
		return bFile.OpenFileReader(hash, isPartial)
	}

	return nil, os.ErrNotExist
}

func (this *FS) RemoveFile(hash string) error {
	// TODO 需要实现
	return nil
}

func (this *FS) Close() error {
	this.isClosed = true

	var lastErr error
	this.mu.Lock()
	for _, bFile := range this.bMap {
		err := bFile.Close()
		if err != nil {
			lastErr = err
		}
	}
	this.mu.Unlock()
	return lastErr
}

func (this *FS) bPathForHash(hash string) (path string, bName string, err error) {
	err = CheckHashErr(hash)
	if err != nil {
		return "", "", err
	}

	return this.dir + "/" + hash[:2] + "/" + hash[2:4] + BFileExt, hash[:4], nil
}

func (this *FS) syncLoop() {
	if this.isClosed {
		return
	}

	if this.opt.SyncTimeout <= 0 {
		return
	}

	var maxSyncFiles = this.opt.MaxSyncFiles
	if maxSyncFiles <= 0 {
		maxSyncFiles = 32
	}

	var bFiles []*BlocksFile

	this.mu.RLock()
	for _, bFile := range this.bMap {
		if time.Since(bFile.SyncAt()) > this.opt.SyncTimeout {
			bFiles = append(bFiles, bFile)
			maxSyncFiles--
			if maxSyncFiles <= 0 {
				break
			}
		}
	}
	this.mu.RUnlock()

	for _, bFile := range bFiles {
		err := bFile.ForceSync()
		if err != nil {
			// TODO 可以在options自定义一个logger
			log.Println("BFS", "sync failed: "+err.Error())
		}
	}
}
