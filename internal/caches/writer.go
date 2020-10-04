package caches

import (
	"os"
	"sync"
)

type Writer struct {
	rawWriter  *os.File
	key        string
	size       int64
	expiredAt  int64
	locker     *sync.RWMutex
	isReleased bool
}

func NewWriter(rawWriter *os.File, key string, expiredAt int64, locker *sync.RWMutex) *Writer {
	return &Writer{
		key:       key,
		rawWriter: rawWriter,
		expiredAt: expiredAt,
		locker:    locker,
	}
}

// 写入数据
func (this *Writer) Write(data []byte) error {
	n, err := this.rawWriter.Write(data)
	this.size += int64(n)
	if err != nil {
		_ = this.rawWriter.Close()
		_ = os.Remove(this.rawWriter.Name())
		this.Release()
	}

	return err
}

// 关闭
func (this *Writer) Close() error {
	// 写入结束符
	_, err := this.rawWriter.WriteString("\n$$$")
	if err != nil {
		_ = os.Remove(this.rawWriter.Name())
	}

	this.Release()

	return err
}

func (this *Writer) Size() int64 {
	return this.size
}

func (this *Writer) ExpiredAt() int64 {
	return this.expiredAt
}

func (this *Writer) Key() string {
	return this.key
}

// 释放锁，一定要调用
func (this *Writer) Release() {
	if this.isReleased {
		return
	}
	this.isReleased = true
	this.locker.Unlock()
}
