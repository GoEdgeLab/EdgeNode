// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package bfs

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/TeaOSLab/EdgeNode/internal/utils/fasttime"
	"github.com/TeaOSLab/EdgeNode/internal/zero"
	"github.com/klauspost/compress/gzip"
	"io"
	"os"
	"sync"
)

const MFileExt = ".m"
const Version1 = 1

type MetaFile struct {
	fp        *os.File
	filename  string
	headerMap map[string]*LazyFileHeader // hash => *LazyFileHeader
	mu        *sync.RWMutex              // TODO 考虑单独一个，不要和bFile共享？

	isModified      bool
	modifiedHashMap map[string]zero.Zero // hash => Zero
}

func OpenMetaFile(filename string, mu *sync.RWMutex) (*MetaFile, error) {
	fp, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return nil, err
	}

	var mFile = &MetaFile{
		filename:        filename,
		fp:              fp,
		headerMap:       map[string]*LazyFileHeader{},
		mu:              mu,
		modifiedHashMap: map[string]zero.Zero{},
	}

	// 从文件中加载已有的文件头信息
	err = mFile.load()
	if err != nil {
		return nil, err
	}

	return mFile, nil
}

func (this *MetaFile) load() error {
	AckReadThread()
	_, err := this.fp.Seek(0, io.SeekStart)
	ReleaseReadThread()
	if err != nil {
		return err
	}

	// TODO 检查文件是否完整

	var buf = make([]byte, 4<<10)
	var blockBytes []byte
	for {
		AckReadThread()
		n, readErr := this.fp.Read(buf)
		ReleaseReadThread()
		if n > 0 {
			blockBytes = append(blockBytes, buf[:n]...)
			for len(blockBytes) > 4 {
				var l = int(binary.BigEndian.Uint32(blockBytes[:4])) + 4 /* Len **/
				if len(blockBytes) < l {
					break
				}

				action, hash, data, decodeErr := DecodeMetaBlock(blockBytes[:l])
				if decodeErr != nil {
					return decodeErr
				}

				switch action {
				case MetaActionNew:
					this.headerMap[hash] = NewLazyFileHeaderFromData(data)
				case MetaActionRemove:
					delete(this.headerMap, hash)
				}

				blockBytes = blockBytes[l:]
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return readErr
		}
	}

	return nil
}

func (this *MetaFile) WriteMeta(hash string, status int, expiresAt int64, expectedFileSize int64) error {

	this.mu.Lock()
	defer this.mu.Unlock()

	this.headerMap[hash] = NewLazyFileHeader(&FileHeader{
		Version:         Version1,
		ExpiresAt:       expiresAt,
		Status:          status,
		ExpiredBodySize: expectedFileSize,
		IsWriting:       true,
	})

	this.modifiedHashMap[hash] = zero.Zero{}

	return nil
}

func (this *MetaFile) WriteHeaderBlockUnsafe(hash string, bOffsetFrom int64, bOffsetTo int64) error {
	lazyHeader, ok := this.headerMap[hash]
	if !ok {
		return nil
	}

	header, err := lazyHeader.FileHeaderUnsafe()
	if err != nil {
		return err
	}

	// TODO 合并相邻block
	header.HeaderBlocks = append(header.HeaderBlocks, BlockInfo{
		BFileOffsetFrom: bOffsetFrom,
		BFileOffsetTo:   bOffsetTo,
	})

	this.modifiedHashMap[hash] = zero.Zero{}

	return nil
}

func (this *MetaFile) WriteBodyBlockUnsafe(hash string, bOffsetFrom int64, bOffsetTo int64, originOffsetFrom int64, originOffsetTo int64) error {
	lazyHeader, ok := this.headerMap[hash]
	if !ok {
		return nil
	}

	header, err := lazyHeader.FileHeaderUnsafe()
	if err != nil {
		return err
	}

	// TODO 合并相邻block
	header.BodyBlocks = append(header.BodyBlocks, BlockInfo{
		OriginOffsetFrom: originOffsetFrom,
		OriginOffsetTo:   originOffsetTo,
		BFileOffsetFrom:  bOffsetFrom,
		BFileOffsetTo:    bOffsetTo,
	})

	this.modifiedHashMap[hash] = zero.Zero{}

	return nil
}

func (this *MetaFile) WriteClose(hash string, headerSize int64, bodySize int64) error {
	// TODO 考虑单个hash多次重复调用的情况

	this.mu.Lock()
	lazyHeader, ok := this.headerMap[hash]
	if !ok {
		this.mu.Unlock()
		return nil
	}

	header, err := lazyHeader.FileHeaderUnsafe()
	if err != nil {
		return err
	}

	this.mu.Unlock()

	// TODO 检查bodySize和expectedBodySize是否一致，如果不一致则从headerMap中删除

	header.ModifiedAt = fasttime.Now().Unix()
	header.HeaderSize = headerSize
	header.BodySize = bodySize
	header.Compact()

	blockBytes, err := this.encodeFileHeader(hash, header)
	if err != nil {
		return err
	}

	this.mu.Lock()
	defer this.mu.Unlock()

	AckReadThread()
	_, err = this.fp.Seek(0, io.SeekEnd)
	ReleaseReadThread()
	if err != nil {
		return err
	}

	AckWriteThread()
	_, err = this.fp.Write(blockBytes)
	ReleaseWriteThread()

	this.isModified = true
	return err
}

func (this *MetaFile) RemoveFile(hash string) error {
	this.mu.Lock()
	defer this.mu.Unlock()

	_, ok := this.headerMap[hash]
	if ok {
		delete(this.headerMap, hash)
	}

	if ok {
		blockBytes, err := EncodeMetaBlock(MetaActionRemove, hash, nil)
		if err != nil {
			return err
		}

		AckWriteThread()
		_, err = this.fp.Write(blockBytes)
		ReleaseWriteThread()
		if err != nil {
			return err
		}
		this.isModified = true
	}

	return nil
}

func (this *MetaFile) FileHeader(hash string) (header *FileHeader, ok bool) {
	this.mu.RLock()
	defer this.mu.RUnlock()

	lazyHeader, ok := this.headerMap[hash]

	if ok {
		var err error
		header, err = lazyHeader.FileHeaderUnsafe()
		if err != nil {
			ok = false
		}
	}
	return
}

func (this *MetaFile) FileHeaderUnsafe(hash string) (header *FileHeader, ok bool) {
	lazyHeader, ok := this.headerMap[hash]

	if ok {
		var err error
		header, err = lazyHeader.FileHeaderUnsafe()
		if err != nil {
			ok = false
		}
	}

	return
}

func (this *MetaFile) CloneFileHeader(hash string) (header *FileHeader, ok bool) {
	this.mu.RLock()
	defer this.mu.RUnlock()
	lazyHeader, ok := this.headerMap[hash]
	if !ok {
		return
	}

	var err error
	header, err = lazyHeader.FileHeaderUnsafe()
	if err != nil {
		ok = false
		return
	}

	header = header.Clone()
	return
}

func (this *MetaFile) FileHeaders() map[string]*LazyFileHeader {
	this.mu.RLock()
	defer this.mu.RUnlock()
	return this.headerMap
}

func (this *MetaFile) ExistFile(hash string) bool {
	this.mu.RLock()
	defer this.mu.RUnlock()

	_, ok := this.headerMap[hash]
	return ok
}

// Compact the meta file
// TODO 考虑自动Compact的时机（脏数据比例？）
func (this *MetaFile) Compact() error {
	this.mu.Lock()
	defer this.mu.Unlock()

	var buf = bytes.NewBuffer(nil)
	for hash, lazyHeader := range this.headerMap {
		header, err := lazyHeader.FileHeaderUnsafe()
		if err != nil {
			return err
		}

		blockBytes, err := this.encodeFileHeader(hash, header)
		if err != nil {
			return err
		}
		buf.Write(blockBytes)
	}

	AckWriteThread()
	err := this.fp.Truncate(int64(buf.Len()))
	ReleaseWriteThread()
	if err != nil {
		return err
	}

	AckReadThread()
	_, err = this.fp.Seek(0, io.SeekStart)
	ReleaseReadThread()
	if err != nil {
		return err
	}

	AckWriteThread()
	_, err = this.fp.Write(buf.Bytes())
	ReleaseWriteThread()
	this.isModified = true
	return err
}

func (this *MetaFile) SyncUnsafe() error {
	if !this.isModified {
		return nil
	}

	AckWriteThread()
	err := this.fp.Sync()
	ReleaseWriteThread()
	if err != nil {
		return err
	}

	for hash := range this.modifiedHashMap {
		lazyHeader, ok := this.headerMap[hash]
		if ok {
			header, decodeErr := lazyHeader.FileHeaderUnsafe()
			if decodeErr != nil {
				return decodeErr
			}
			header.IsWriting = false
		}
	}

	this.isModified = false
	this.modifiedHashMap = map[string]zero.Zero{}
	return nil
}

// Close 关闭当前文件
func (this *MetaFile) Close() error {
	return this.fp.Close()
}

// RemoveAll 删除所有数据
func (this *MetaFile) RemoveAll() error {
	_ = this.fp.Close()
	return os.Remove(this.fp.Name())
}

// encode file header to data bytes
func (this *MetaFile) encodeFileHeader(hash string, header *FileHeader) ([]byte, error) {
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return nil, err
	}

	var buf = utils.SharedBufferPool.Get()
	defer utils.SharedBufferPool.Put(buf)

	// TODO 考虑使用gzip pool
	gzWriter, err := gzip.NewWriterLevel(buf, gzip.BestSpeed)
	if err != nil {
		return nil, err
	}

	_, err = gzWriter.Write(headerJSON)
	if err != nil {
		_ = gzWriter.Close()
		return nil, err
	}

	err = gzWriter.Close()
	if err != nil {
		return nil, err
	}

	return EncodeMetaBlock(MetaActionNew, hash, buf.Bytes())
}
