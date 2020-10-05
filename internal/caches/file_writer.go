package caches

import (
	"os"
	"sync"
)

type FileWriter struct {
	rawWriter  *os.File
	key        string
	size       int64
	expiredAt  int64
	locker     *sync.RWMutex
	isReleased bool
}

func NewFileWriter(rawWriter *os.File, key string, expiredAt int64, locker *sync.RWMutex) *FileWriter {
	return &FileWriter{
		key:       key,
		rawWriter: rawWriter,
		expiredAt: expiredAt,
		locker:    locker,
	}
}

// 写入数据
func (this *FileWriter) Write(data []byte) (n int, err error) {
	n, err = this.rawWriter.Write(data)
	this.size += int64(n)
	if err != nil {
		_ = this.rawWriter.Close()
		_ = os.Remove(this.rawWriter.Name())
		this.Release()
	}
	return
}

// 关闭
func (this *FileWriter) Close() error {
	// 写入结束符
	_, err := this.rawWriter.WriteString("\n$$$")
	if err != nil {
		_ = os.Remove(this.rawWriter.Name())
	}

	this.Release()

	return err
}

// 丢弃
func (this *FileWriter) Discard() error {
	err := os.Remove(this.rawWriter.Name())
	this.Release()
	return err
}

func (this *FileWriter) Size() int64 {
	return this.size
}

func (this *FileWriter) ExpiredAt() int64 {
	return this.expiredAt
}

func (this *FileWriter) Key() string {
	return this.key
}

// 释放锁，一定要调用
func (this *FileWriter) Release() {
	if this.isReleased {
		return
	}
	this.isReleased = true
	this.locker.Unlock()
}
