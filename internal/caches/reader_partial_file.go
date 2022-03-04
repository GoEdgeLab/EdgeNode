package caches

import (
	"encoding/binary"
	"errors"
	rangeutils "github.com/TeaOSLab/EdgeNode/internal/utils/ranges"
	"github.com/iwind/TeaGo/types"
	"io"
	"os"
)

type PartialFileReader struct {
	*FileReader

	ranges    *PartialRanges
	rangePath string
}

func NewPartialFileReader(fp *os.File) *PartialFileReader {
	return &PartialFileReader{
		FileReader: NewFileReader(fp),
		rangePath:  partialRangesFilePath(fp.Name()),
	}
}

func (this *PartialFileReader) Init() error {
	return this.InitAutoDiscard(true)
}

func (this *PartialFileReader) InitAutoDiscard(autoDiscard bool) error {
	if this.openFile != nil {
		this.meta = this.openFile.meta
		this.header = this.openFile.header
	}

	isOk := false

	if autoDiscard {
		defer func() {
			if !isOk {
				_ = this.discard()
			}
		}()
	}

	// 读取Range
	ranges, err := NewPartialRangesFromFile(this.rangePath)
	if err != nil {
		return errors.New("read ranges failed: " + err.Error())
	}
	this.ranges = ranges

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

	status := types.Int(string(buf[SizeExpiresAt : SizeExpiresAt+SizeStatus]))
	if status < 100 || status > 999 {
		return errors.New("invalid status")
	}
	this.status = status

	// URL
	urlLength := binary.BigEndian.Uint32(buf[SizeExpiresAt+SizeStatus : SizeExpiresAt+SizeStatus+SizeURLLength])

	// header
	headerSize := int(binary.BigEndian.Uint32(buf[SizeExpiresAt+SizeStatus+SizeURLLength : SizeExpiresAt+SizeStatus+SizeURLLength+SizeHeaderLength]))
	if headerSize == 0 {
		return nil
	}
	this.headerSize = headerSize
	this.headerOffset = int64(SizeMeta) + int64(urlLength)

	// body
	this.bodyOffset = this.headerOffset + int64(headerSize)
	bodySize := int(binary.BigEndian.Uint64(buf[SizeExpiresAt+SizeStatus+SizeURLLength+SizeHeaderLength : SizeExpiresAt+SizeStatus+SizeURLLength+SizeHeaderLength+SizeBodyLength]))
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

// ContainsRange 是否包含某些区间内容
// 这里的 r 是已经经过格式化的
func (this *PartialFileReader) ContainsRange(r rangeutils.Range) bool {
	return this.ranges.Contains(r.Start(), r.End())
}

// MaxLength 获取区间最大长度
func (this *PartialFileReader) MaxLength() int64 {
	if this.bodySize > 0 {
		return this.bodySize
	}
	return this.ranges.Max() + 1
}

func (this *PartialFileReader) discard() error {
	_ = os.Remove(this.rangePath)
	return this.FileReader.discard()
}
