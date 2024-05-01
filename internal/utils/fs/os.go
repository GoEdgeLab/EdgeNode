// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package fsutils

import (
	"os"
)

func Remove(filename string) (err error) {
	WriterLimiter.Ack()
	err = os.Remove(filename)
	WriterLimiter.Release()
	return
}

func Rename(oldPath string, newPath string) (err error) {
	WriterLimiter.Ack()
	err = os.Rename(oldPath, newPath)
	WriterLimiter.Release()
	return
}

func ReadFile(filename string) (data []byte, err error) {
	ReaderLimiter.Ack()
	data, err = os.ReadFile(filename)
	ReaderLimiter.Release()
	return
}

func WriteFile(filename string, data []byte, perm os.FileMode) (err error) {
	WriterLimiter.Ack()
	err = os.WriteFile(filename, data, perm)
	WriterLimiter.Release()
	return
}

func OpenFile(name string, flag int, perm os.FileMode) (f *os.File, err error) {
	if flag&os.O_RDONLY == os.O_RDONLY {
		ReaderLimiter.Ack()
	}

	f, err = os.OpenFile(name, flag, perm)

	if flag&os.O_RDONLY == os.O_RDONLY {
		ReaderLimiter.Release()
	}

	return
}

func Open(name string) (f *os.File, err error) {
	ReaderLimiter.Ack()
	f, err = os.Open(name)
	ReaderLimiter.Release()
	return
}
