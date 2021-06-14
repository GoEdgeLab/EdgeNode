package caches

import (
	"errors"
)

type MemoryReader struct {
	item *MemoryItem
}

func NewMemoryReader(item *MemoryItem) *MemoryReader {
	return &MemoryReader{item: item}
}

func (this *MemoryReader) Init() error {
	return nil
}

func (this *MemoryReader) TypeName() string {
	return "memory"
}

func (this *MemoryReader) Status() int {
	return this.item.Status
}

func (this *MemoryReader) LastModified() int64 {
	return this.item.ModifiedAt
}

func (this *MemoryReader) HeaderSize() int64 {
	return int64(len(this.item.HeaderValue))
}

func (this *MemoryReader) BodySize() int64 {
	return int64(len(this.item.BodyValue))
}

func (this *MemoryReader) ReadHeader(buf []byte, callback ReaderFunc) error {
	l := len(buf)
	if l == 0 {
		return errors.New("using empty buffer")
	}

	size := len(this.item.HeaderValue)
	offset := 0
	for {
		left := size - offset
		if l <= left {
			copy(buf, this.item.HeaderValue[offset:offset+l])
			goNext, e := callback(l)
			if e != nil {
				return e
			}
			if !goNext {
				break
			}
		} else {
			copy(buf, this.item.HeaderValue[offset:])
			_, e := callback(left)
			if e != nil {
				return e
			}
			break
		}
		offset += l
		if offset >= size {
			break
		}
	}

	return nil
}

func (this *MemoryReader) ReadBody(buf []byte, callback ReaderFunc) error {
	l := len(buf)
	if l == 0 {
		return errors.New("using empty buffer")
	}

	size := len(this.item.BodyValue)
	offset := 0
	for {
		left := size - offset
		if l <= left {
			copy(buf, this.item.BodyValue[offset:offset+l])
			goNext, e := callback(l)
			if e != nil {
				return e
			}
			if !goNext {
				break
			}
		} else {
			copy(buf, this.item.BodyValue[offset:])
			_, e := callback(left)
			if e != nil {
				return e
			}
			break
		}
		offset += l
		if offset >= size {
			break
		}
	}
	return nil
}

func (this *MemoryReader) ReadBodyRange(buf []byte, start int64, end int64, callback ReaderFunc) error {
	offset := start
	bodySize := int64(len(this.item.BodyValue))
	if start < 0 {
		offset = bodySize + end
		end = bodySize - 1
	} else if end < 0 {
		offset = start
		end = bodySize - 1
	}

	if end >= bodySize {
		end = bodySize - 1
	}

	if offset < 0 || end < 0 || offset > end {
		return ErrInvalidRange
	}

	newData := this.item.BodyValue[offset : end+1]

	l := len(buf)
	if l == 0 {
		return errors.New("using empty buffer")
	}

	size := len(newData)
	offset2 := 0
	for {
		left := size - offset2
		if l <= left {
			copy(buf, newData[offset2:offset2+l])
			goNext, e := callback(l)
			if e != nil {
				return e
			}
			if !goNext {
				break
			}
		} else {
			copy(buf, newData[offset2:])
			_, e := callback(left)
			if e != nil {
				return e
			}
			break
		}
		offset2 += l
		if offset2 >= size {
			break
		}
	}

	return nil
}

func (this *MemoryReader) Close() error {
	return nil
}
