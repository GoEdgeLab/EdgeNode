// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package fsutils

import "os"

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
