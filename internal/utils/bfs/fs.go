// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package bfs

import (
	"errors"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	fsutils "github.com/TeaOSLab/EdgeNode/internal/utils/fs"
	"github.com/TeaOSLab/EdgeNode/internal/utils/linkedlist"
	"github.com/TeaOSLab/EdgeNode/internal/zero"
	"log"
	"runtime"
	"sync"
	"time"
)

func IsEnabled() bool {
	return runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64"
}

// FS 文件系统对象
type FS struct {
	dir string
	opt *FSOptions

	bMap         map[string]*BlocksFile   // name => *BlocksFile
	bList        *linkedlist.List[string] // [bName]
	bItemMap     map[string]*linkedlist.Item[string]
	closingBMap  map[string]zero.Zero // filename => Zero
	closingBChan chan *BlocksFile

	mu       *sync.RWMutex
	isClosed bool

	syncTicker *time.Ticker

	locker *fsutils.Locker
}

// OpenFS 打开文件系统
func OpenFS(dir string, options *FSOptions) (*FS, error) {
	if !IsEnabled() {
		return nil, errors.New("the fs only works under 64 bit system")
	}

	if options == nil {
		options = DefaultFSOptions
	} else {
		options.EnsureDefaults()
	}

	var locker = fsutils.NewLocker(dir + "/fs")
	err := locker.Lock()
	if err != nil {
		return nil, err
	}

	var fs = &FS{
		dir:          dir,
		bMap:         map[string]*BlocksFile{},
		bList:        linkedlist.NewList[string](),
		bItemMap:     map[string]*linkedlist.Item[string]{},
		closingBMap:  map[string]zero.Zero{},
		closingBChan: make(chan *BlocksFile, 32),
		mu:           &sync.RWMutex{},
		opt:          options,
		syncTicker:   time.NewTicker(1 * time.Second),
		locker:       locker,
	}
	go fs.init()
	return fs, nil
}

func (this *FS) init() {
	go func() {
		// sync in background
		for range this.syncTicker.C {
			this.syncLoop()
		}
	}()

	go func() {
		for {
			this.processClosingBFiles()
		}
	}()
}

// OpenFileWriter 打开文件写入器
func (this *FS) OpenFileWriter(hash string, bodySize int64, isPartial bool) (*FileWriter, error) {
	if this.isClosed {
		return nil, errors.New("the fs closed")
	}

	if isPartial && bodySize <= 0 {
		return nil, errors.New("invalid body size for partial content")
	}

	bFile, err := this.openBFileForHashWriting(hash)
	if err != nil {
		return nil, err
	}
	return bFile.OpenFileWriter(hash, bodySize, isPartial)
}

// OpenFileReader 打开文件读取器
func (this *FS) OpenFileReader(hash string, isPartial bool) (*FileReader, error) {
	if this.isClosed {
		return nil, errors.New("the fs closed")
	}

	bFile, err := this.openBFileForHashReading(hash)
	if err != nil {
		return nil, err
	}
	return bFile.OpenFileReader(hash, isPartial)
}

func (this *FS) ExistFile(hash string) (bool, error) {
	if this.isClosed {
		return false, errors.New("the fs closed")
	}

	bFile, err := this.openBFileForHashReading(hash)
	if err != nil {
		return false, err
	}
	return bFile.ExistFile(hash), nil
}

func (this *FS) RemoveFile(hash string) error {
	if this.isClosed {
		return errors.New("the fs closed")
	}

	bFile, err := this.openBFileForHashWriting(hash)
	if err != nil {
		return err
	}
	return bFile.RemoveFile(hash)
}

func (this *FS) Close() error {
	if this.isClosed {
		return nil
	}

	this.isClosed = true

	close(this.closingBChan)
	this.syncTicker.Stop()

	var lastErr error
	this.mu.Lock()
	if len(this.bMap) > 0 {
		var g = goman.NewTaskGroup()
		for _, bFile := range this.bMap {
			var bFileCopy = bFile
			g.Run(func() {
				err := bFileCopy.Close()
				if err != nil {
					lastErr = err
				}
			})
		}

		g.Wait()
	}
	this.mu.Unlock()

	err := this.locker.Release()
	if err != nil {
		lastErr = err
	}

	return lastErr
}

func (this *FS) TestBMap() map[string]*BlocksFile {
	return this.bMap
}

func (this *FS) TestBList() *linkedlist.List[string] {
	return this.bList
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
		if bFile.IsClosing() {
			continue
		}

		err := bFile.ForceSync()
		if err != nil {
			// check again
			if bFile.IsClosing() {
				continue
			}

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
		// 调整当前BFile所在位置
		this.mu.Lock()

		if bFile.IsClosing() {
			// TODO 需要重新等待打开
		}

		item, itemOk := this.bItemMap[bName]
		if itemOk {
			this.bList.Remove(item)
			this.bList.Push(item)
		}
		this.mu.Unlock()

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

	err = this.waitBFile(bPath)
	if err != nil {
		return nil, err
	}

	this.mu.Lock()
	bFile, ok := this.bMap[bName]
	if ok {
		// 调整当前BFile所在位置
		item, itemOk := this.bItemMap[bName]
		if itemOk {
			this.bList.Remove(item)
			this.bList.Push(item)
		}
		this.mu.Unlock()
		return bFile, nil
	}

	this.mu.Unlock()

	return this.openBFile(bPath, bName)
}

func (this *FS) openBFile(bPath string, bName string) (*BlocksFile, error) {
	// check closing queue
	err := this.waitBFile(bPath)
	if err != nil {
		return nil, err
	}

	this.mu.Lock()
	defer this.mu.Unlock()

	// lookup again
	bFile, ok := this.bMap[bName]
	if ok {
		return bFile, nil
	}

	// TODO 不要把 OpenBlocksFile 放入到 mu 中？
	bFile, err = OpenBlocksFile(bPath, &BlockFileOptions{
		BytesPerSync: this.opt.BytesPerSync,
	})
	if err != nil {
		return nil, err
	}

	// 防止被关闭
	bFile.IncrRef()
	defer bFile.DecrRef()

	this.bMap[bName] = bFile

	// 加入到列表中
	var item = linkedlist.NewItem(bName)
	this.bList.Push(item)
	this.bItemMap[bName] = item

	// 检查是否超出maxOpenFiles
	if this.bList.Len() > this.opt.MaxOpenFiles {
		this.shiftOpenFiles()
	}

	return bFile, nil
}

// 处理关闭中的 BFile 们
func (this *FS) processClosingBFiles() {
	if this.isClosed {
		return
	}

	var bFile = <-this.closingBChan
	if bFile == nil {
		return
	}

	_ = bFile.Close()

	this.mu.Lock()
	delete(this.closingBMap, bFile.Filename())
	this.mu.Unlock()
}

// 弹出超出BFile数量限制的BFile
func (this *FS) shiftOpenFiles() {
	var l = this.bList.Len()
	var count = l - this.opt.MaxOpenFiles
	if count <= 0 {
		return
	}

	var bNames []string
	var searchCount int
	this.bList.Range(func(item *linkedlist.Item[string]) (goNext bool) {
		searchCount++

		var bName = item.Value
		var bFile = this.bMap[bName]
		if bFile.CanClose() {
			bNames = append(bNames, bName)
			count--
		}
		return count > 0 && searchCount < 8 && searchCount < l-8
	})

	for _, bName := range bNames {
		var bFile = this.bMap[bName]
		var item = this.bItemMap[bName]

		// clean
		delete(this.bMap, bName)
		delete(this.bItemMap, bName)
		this.bList.Remove(item)

		// add to closing queue
		this.closingBMap[bFile.Filename()] = zero.Zero{}

		// MUST run in goroutine
		go func(bFile *BlocksFile) {
			// 因为 closingBChan 可能已经关闭
			defer func() {
				recover()
			}()

			this.closingBChan <- bFile
		}(bFile)
	}
}

func (this *FS) waitBFile(bPath string) error {
	this.mu.RLock()
	_, isClosing := this.closingBMap[bPath]
	this.mu.RUnlock()
	if !isClosing {
		return nil
	}

	var maxWaits = 30_000
	for {
		this.mu.RLock()
		_, isClosing = this.closingBMap[bPath]
		this.mu.RUnlock()
		if !isClosing {
			break
		}
		time.Sleep(1 * time.Millisecond)
		maxWaits--

		if maxWaits < 0 {
			return errors.New("open blocks file timeout")
		}
	}
	return nil
}
