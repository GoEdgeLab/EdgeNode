package caches

import (
	"encoding/binary"
	"errors"
	rangeutils "github.com/TeaOSLab/EdgeNode/internal/utils/ranges"
	"github.com/iwind/TeaGo/types"
	"io"
	"os"
)

type FileReader struct {
	fp *os.File

	openFile      *OpenFile
	openFileCache *OpenFileCache

	meta   []byte
	header []byte

	expiresAt    int64
	status       int
	headerOffset int64
	headerSize   int
	bodySize     int64
	bodyOffset   int64

	isClosed bool
}

func NewFileReader(fp *os.File) *FileReader {
	return &FileReader{fp: fp}
}

func (this *FileReader) Init() error {
	return this.InitAutoDiscard(true)
}

func (this *FileReader) InitAutoDiscard(autoDiscard bool) error {
	if this.openFile != nil {
		this.meta = this.openFile.meta
		this.header = this.openFile.header
	}

	var isOk = false

	if autoDiscard {
		defer func() {
			if !isOk {
				_ = this.discard()
			}
		}()
	}

	var buf = this.meta
	if len(buf) == 0 {
		buf = make([]byte, SizeMeta)
		ok, err := this.readToBuff(this.fp, buf)
		if err != nil {
			return err
		}
		if !ok {
			return ErrNotFound
		}
		this.meta = buf
	}

	this.expiresAt = int64(binary.BigEndian.Uint32(buf[:SizeExpiresAt]))

	var status = types.Int(string(buf[OffsetStatus : OffsetStatus+SizeStatus]))
	if status < 100 || status > 999 {
		return errors.New("invalid status")
	}
	this.status = status

	// URL
	var urlLength = binary.BigEndian.Uint32(buf[OffsetURLLength : OffsetURLLength+SizeURLLength])

	// header
	var headerSize = int(binary.BigEndian.Uint32(buf[OffsetHeaderLength : OffsetHeaderLength+SizeHeaderLength]))
	if headerSize == 0 {
		return nil
	}
	this.headerSize = headerSize
	this.headerOffset = int64(SizeMeta) + int64(urlLength)

	// body
	this.bodyOffset = this.headerOffset + int64(headerSize)
	var bodySize = int(binary.BigEndian.Uint64(buf[OffsetBodyLength : OffsetBodyLength+SizeBodyLength]))
	if bodySize == 0 {
		isOk = true
		return nil
	}
	this.bodySize = int64(bodySize)

	// read header
	if this.openFileCache != nil && len(this.header) == 0 {
		if headerSize > 0 && headerSize <= 512 {
			this.header = make([]byte, headerSize)
			_, err := this.fp.Seek(this.headerOffset, io.SeekStart)
			if err != nil {
				return err
			}
			_, err = this.readToBuff(this.fp, this.header)
			if err != nil {
				return err
			}
		}
	}

	isOk = true

	return nil
}

func (this *FileReader) TypeName() string {
	return "disk"
}

func (this *FileReader) ExpiresAt() int64 {
	return this.expiresAt
}

func (this *FileReader) Status() int {
	return this.status
}

func (this *FileReader) LastModified() int64 {
	stat, err := this.fp.Stat()
	if err != nil {
		return 0
	}
	return stat.ModTime().Unix()
}

func (this *FileReader) HeaderSize() int64 {
	return int64(this.headerSize)
}

func (this *FileReader) BodySize() int64 {
	return this.bodySize
}

func (this *FileReader) ReadHeader(buf []byte, callback ReaderFunc) error {
	// 使用缓存
	if len(this.header) > 0 && len(buf) >= len(this.header) {
		copy(buf, this.header)
		_, err := callback(len(this.header))
		if err != nil {
			return err
		}

		// 移动到Body位置
		_, err = this.fp.Seek(this.bodyOffset, io.SeekStart)
		if err != nil {
			return err
		}
		return nil
	}

	var isOk = false

	defer func() {
		if !isOk {
			_ = this.discard()
		}
	}()

	_, err := this.fp.Seek(this.headerOffset, io.SeekStart)
	if err != nil {
		return err
	}

	var headerSize = this.headerSize

	for {
		n, err := this.fp.Read(buf)
		if n > 0 {
			if n < headerSize {
				goNext, e := callback(n)
				if e != nil {
					isOk = true
					return e
				}
				if !goNext {
					break
				}
				headerSize -= n
			} else {
				_, e := callback(headerSize)
				if e != nil {
					isOk = true
					return e
				}
				break
			}
		}
		if err != nil {
			if err != io.EOF {
				return err
			}
			break
		}
	}

	isOk = true

	// 移动到Body位置
	_, err = this.fp.Seek(this.bodyOffset, io.SeekStart)
	if err != nil {
		return err
	}

	return nil
}

func (this *FileReader) ReadBody(buf []byte, callback ReaderFunc) error {
	if this.bodySize == 0 {
		return nil
	}

	var isOk = false

	defer func() {
		if !isOk {
			_ = this.discard()
		}
	}()

	var offset = this.bodyOffset

	// 开始读Body部分
	_, err := this.fp.Seek(offset, io.SeekStart)
	if err != nil {
		return err
	}

	for {
		n, err := this.fp.Read(buf)
		if n > 0 {
			goNext, e := callback(n)
			if e != nil {
				isOk = true
				return e
			}
			if !goNext {
				break
			}
		}
		if err != nil {
			if err != io.EOF {
				return err
			}
			break
		}
	}

	isOk = true

	return nil
}

func (this *FileReader) Read(buf []byte) (n int, err error) {
	if this.bodySize == 0 {
		n = 0
		err = io.EOF
		return
	}

	n, err = this.fp.Read(buf)
	if err != nil && err != io.EOF {
		_ = this.discard()
	}

	return
}

func (this *FileReader) ReadBodyRange(buf []byte, start int64, end int64, callback ReaderFunc) error {
	var isOk = false

	defer func() {
		if !isOk {
			_ = this.discard()
		}
	}()

	var offset = start
	if start < 0 {
		offset = this.bodyOffset + this.bodySize + end
		end = this.bodyOffset + this.bodySize - 1
	} else if end < 0 {
		offset = this.bodyOffset + start
		end = this.bodyOffset + this.bodySize - 1
	} else {
		offset = this.bodyOffset + start
		end = this.bodyOffset + end
	}
	if offset < 0 || end < 0 || offset > end {
		isOk = true
		return ErrInvalidRange
	}
	_, err := this.fp.Seek(offset, io.SeekStart)
	if err != nil {
		return err
	}

	for {
		n, err := this.fp.Read(buf)
		if n > 0 {
			var n2 = int(end-offset) + 1
			if n2 <= n {
				_, e := callback(n2)
				if e != nil {
					isOk = true
					return e
				}
				break
			} else {
				goNext, e := callback(n)
				if e != nil {
					isOk = true
					return e
				}
				if !goNext {
					break
				}
			}

			offset += int64(n)
			if offset > end {
				break
			}
		}
		if err != nil {
			if err != io.EOF {
				return err
			}
			break
		}
	}

	isOk = true

	return nil
}

// ContainsRange 是否包含某些区间内容
func (this *FileReader) ContainsRange(r rangeutils.Range) (r2 rangeutils.Range, ok bool) {
	return r, true
}

// FP 原始的文件句柄
func (this *FileReader) FP() *os.File {
	return this.fp
}

func (this *FileReader) Close() error {
	if this.isClosed {
		return nil
	}
	this.isClosed = true

	if this.openFileCache != nil {
		if this.openFile != nil {
			this.openFileCache.Put(this.fp.Name(), this.openFile)
		} else {
			var cacheMeta = make([]byte, len(this.meta))
			copy(cacheMeta, this.meta)
			this.openFileCache.Put(this.fp.Name(), NewOpenFile(this.fp, cacheMeta, this.header, this.LastModified(), this.bodySize))
		}
		return nil
	}

	return this.fp.Close()
}

func (this *FileReader) readToBuff(fp *os.File, buf []byte) (ok bool, err error) {
	n, err := fp.Read(buf)
	if err != nil {
		return false, err
	}
	ok = n == len(buf)
	return
}

func (this *FileReader) discard() error {
	_ = this.fp.Close()
	this.isClosed = true

	// close open file cache
	if this.openFileCache != nil {
		this.openFileCache.Close(this.fp.Name())
	}

	// remove file
	return os.Remove(this.fp.Name())
}
