package caches

import (
	"encoding/binary"
	"errors"
	"github.com/iwind/TeaGo/types"
	"io"
	"os"
)

type FileReader struct {
	fp *os.File

	status       int
	headerOffset int64
	headerSize   int
	bodySize     int64
	bodyOffset   int64

	bodyBufLen int
	bodyBuf    []byte
}

func NewFileReader(fp *os.File) *FileReader {
	return &FileReader{fp: fp}
}

func (this *FileReader) Init() error {
	isOk := false

	defer func() {
		if !isOk {
			_ = this.discard()
		}
	}()

	// 读取状态
	_, err := this.fp.Seek(SizeExpiresAt, io.SeekStart)
	if err != nil {
		_ = this.discard()
		return err
	}
	buf := make([]byte, 3)
	ok, err := this.readToBuff(this.fp, buf)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	status := types.Int(string(buf))
	if status < 100 || status > 999 {
		return errors.New("invalid status")
	}
	this.status = status

	// URL
	_, err = this.fp.Seek(SizeExpiresAt+SizeStatus, io.SeekStart)
	if err != nil {
		return err
	}

	bytes4 := make([]byte, 4)
	ok, err = this.readToBuff(this.fp, bytes4)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	urlLength := binary.BigEndian.Uint32(bytes4)

	// header
	ok, err = this.readToBuff(this.fp, bytes4)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	headerSize := int(binary.BigEndian.Uint32(bytes4))
	if headerSize == 0 {
		return nil
	}
	this.headerSize = headerSize
	this.headerOffset = int64(SizeMeta) + int64(urlLength)

	// body
	bytes8 := make([]byte, 8)
	ok, err = this.readToBuff(this.fp, bytes8)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	bodySize := int(binary.BigEndian.Uint64(bytes8))
	if bodySize == 0 {
		return nil
	}
	this.bodySize = int64(bodySize)
	this.bodyOffset = this.headerOffset + int64(headerSize)

	isOk = true

	return nil
}

func (this *FileReader) TypeName() string {
	return "disk"
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
	isOk := false

	defer func() {
		if !isOk {
			_ = this.discard()
		}
	}()

	_, err := this.fp.Seek(this.headerOffset, io.SeekStart)
	if err != nil {
		return err
	}

	headerSize := this.headerSize

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
				if n > headerSize {
					this.bodyBuf = buf[headerSize:]
					this.bodyBufLen = n - headerSize
				}
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

	return nil
}

func (this *FileReader) ReadBody(buf []byte, callback ReaderFunc) error {
	isOk := false

	defer func() {
		if !isOk {
			_ = this.discard()
		}
	}()

	offset := this.bodyOffset

	// 直接返回从Header中剩余的
	if this.bodyBufLen > 0 && len(buf) >= this.bodyBufLen {
		offset += int64(this.bodyBufLen)

		copy(buf, this.bodyBuf)
		isOk = true

		goNext, err := callback(this.bodyBufLen)
		if err != nil {
			return err
		}
		if !goNext {
			return nil
		}

		if this.bodySize <= int64(this.bodyBufLen) {
			return nil
		}
	}

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

func (this *FileReader) ReadBodyRange(buf []byte, start int64, end int64, callback ReaderFunc) error {
	isOk := false

	defer func() {
		if !isOk {
			_ = this.discard()
		}
	}()

	offset := start
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
			n2 := int(end-offset) + 1
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

func (this *FileReader) Close() error {
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
	return os.Remove(this.fp.Name())
}
