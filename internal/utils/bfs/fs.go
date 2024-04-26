// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package bfs

import (
	"errors"
	fsutils "github.com/TeaOSLab/EdgeNode/internal/utils/fs"
	"log"
	"sync"
	"time"
)

// FS 文件系统管理
type FS struct {
	dir string
	opt *FSOptions

	bMap     map[string]*BlocksFile // name => *BlocksFile
	mu       *sync.RWMutex
	isClosed bool

	syncTicker *time.Ticker

	locker *fsutils.Locker
}

func OpenFS(dir string, options *FSOptions) (*FS, error) {
	options.EnsureDefaults()

	var locker = fsutils.NewLocker(dir + "/fs")
	err := locker.Lock()
	if err != nil {
		return nil, err
	}

	var fs = &FS{
		dir:        dir,
		bMap:       map[string]*BlocksFile{},
		mu:         &sync.RWMutex{},
		opt:        options,
		syncTicker: time.NewTicker(1 * time.Second),
		locker:     locker,
	}
	go fs.init()
	return fs, nil
}

func (this *FS) init() {
	// sync in background
	for range this.syncTicker.C {
		this.syncLoop()
	}
}

func (this *FS) OpenFileWriter(hash string, bodySize int64, isPartial bool) (*FileWriter, error) {
	if isPartial && bodySize <= 0 {
		return nil, errors.New("invalid body size for partial content")
	}

	bFile, err := this.openBFileForHashWriting(hash)
	if err != nil {
		return nil, err
	}
	return bFile.OpenFileWriter(hash, bodySize, isPartial)
}

func (this *FS) OpenFileReader(hash string, isPartial bool) (*FileReader, error) {
	bFile, err := this.openBFileForHashReading(hash)
	if err != nil {
		return nil, err
	}
	return bFile.OpenFileReader(hash, isPartial)
}

func (this *FS) ExistFile(hash string) (bool, error) {
	bFile, err := this.openBFileForHashReading(hash)
	if err != nil {
		return false, err
	}
	return bFile.ExistFile(hash), nil
}

func (this *FS) RemoveFile(hash string) error {
	bFile, err := this.openBFileForHashWriting(hash)
	if err != nil {
		return err
	}
	return bFile.RemoveFile(hash)
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

	err := this.locker.Release()
	if err != nil {
		lastErr = err
	}

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

func (this *FS) openBFileForHashWriting(hash string) (*BlocksFile, error) {
	err := CheckHashErr(hash)
	if err != nil {
		return nil, err
	}

	bPath, bName, err := this.bPathForHash(hash)
	if err != nil {
		return nil, err
	}

	this.mu.RLock()
	bFile, ok := this.bMap[bName]
	this.mu.RUnlock()
	if ok {
		return bFile, nil
	}

	return this.openBFile(bPath, bName)
}

func (this *FS) openBFileForHashReading(hash string) (*BlocksFile, error) {
	err := CheckHashErr(hash)
	if err != nil {
		return nil, err
	}

	bPath, bName, err := this.bPathForHash(hash)
	if err != nil {
		return nil, err
	}

	this.mu.RLock()
	bFile, ok := this.bMap[bName]
	this.mu.RUnlock()
	if ok {
		return bFile, nil
	}

	return this.openBFile(bPath, bName)
}

func (this *FS) openBFile(bPath string, bName string) (*BlocksFile, error) {
	this.mu.Lock()
	defer this.mu.Unlock()

	// lookup again
	bFile, ok := this.bMap[bName]
	if ok {
		return bFile, nil
	}

	bFile, err := OpenBlocksFile(bPath, &BlockFileOptions{
		BytesPerSync: this.opt.BytesPerSync,
	})
	if err != nil {
		return nil, err
	}
	this.bMap[bName] = bFile
	return bFile, nil
}
