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
	headerMap map[string]*FileHeader // hash => *FileHeader
	mu        *sync.RWMutex          // TODO 考虑单独一个，不要和bFile共享？

	isModified      bool
	modifiedHashMap map[string]zero.Zero
}

func OpenMetaFile(filename string, mu *sync.RWMutex) (*MetaFile, error) {
	fp, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return nil, err
	}

	var mFile = &MetaFile{
		filename:        filename,
		fp:              fp,
		headerMap:       map[string]*FileHeader{},
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
	_, err := this.fp.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}

	// TODO 考虑文件最后一行未写完整的情形

	var buf = make([]byte, 4<<10)
	var blockBytes []byte
	for {
		n, readErr := this.fp.Read(buf)
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
					header, decodeHeaderErr := this.decodeHeader(data)
					if decodeHeaderErr != nil {
						return decodeHeaderErr
					}
					this.headerMap[hash] = header
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

	this.headerMap[hash] = &FileHeader{
		Version:         Version1,
		ExpiresAt:       expiresAt,
		Status:          status,
		ExpiredBodySize: expectedFileSize,
		IsWriting:       true,
	}

	this.modifiedHashMap[hash] = zero.Zero{}

	return nil
}

func (this *MetaFile) WriteHeaderBlockUnsafe(hash string, bOffsetFrom int64, bOffsetTo int64) error {
	header, ok := this.headerMap[hash]
	if !ok {
		return nil
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
	header, ok := this.headerMap[hash]
	if !ok {
		return nil
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
	header, ok := this.headerMap[hash]
	if ok {
		// TODO 检查bodySize和expectedBodySize是否一致，如果不一致则从headerMap中删除

		header.ModifiedAt = fasttime.Now().Unix()
		header.HeaderSize = headerSize
		header.BodySize = bodySize
		header.Compact()
	}
	this.mu.Unlock()
	if !ok {
		return nil
	}

	blockBytes, err := this.encodeHeader(hash, header)
	if err != nil {
		return err
	}

	this.mu.Lock()
	defer this.mu.Unlock()

	_, err = this.fp.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}

	_, err = this.fp.Write(blockBytes)
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

		_, err = this.fp.Write(blockBytes)
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
	header, ok = this.headerMap[hash]
	return
}

func (this *MetaFile) CloneFileHeader(hash string) (header *FileHeader, ok bool) {
	this.mu.RLock()
	defer this.mu.RUnlock()
	header, ok = this.headerMap[hash]
	if !ok {
		return
	}

	header = header.Clone()
	return
}

func (this *MetaFile) FileHeaders() map[string]*FileHeader {
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
	for hash, header := range this.headerMap {
		blockBytes, err := this.encodeHeader(hash, header)
		if err != nil {
			return err
		}
		buf.Write(blockBytes)
	}

	err := this.fp.Truncate(int64(buf.Len()))
	if err != nil {
		return err
	}

	_, err = this.fp.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}

	_, err = this.fp.Write(buf.Bytes())
	this.isModified = true
	return err
}

func (this *MetaFile) SyncUnsafe() error {
	if !this.isModified {
		return nil
	}

	err := this.fp.Sync()
	if err != nil {
		return err
	}

	for hash := range this.modifiedHashMap {
		header, ok := this.headerMap[hash]
		if ok {
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

func (this *MetaFile) encodeHeader(hash string, header *FileHeader) ([]byte, error) {
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

func (this *MetaFile) decodeHeader(data []byte) (*FileHeader, error) {
	gzReader, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = gzReader.Close()
	}()

	var resultBuf = bytes.NewBuffer(nil)

	var buf = make([]byte, 4096)
	for {
		n, readErr := gzReader.Read(buf)
		if n > 0 {
			resultBuf.Write(buf[:n])
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return nil, readErr
		}
	}

	var header = &FileHeader{}
	err = json.Unmarshal(resultBuf.Bytes(), header)
	if err != nil {
		return nil, err
	}

	header.IsWriting = false
	return header, nil
}
