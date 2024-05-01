// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package fsutils

import "os"

const FlagRead = 0x1
const FlagWrite = 0x2

type File struct {
	rawFile  *os.File
	readonly bool
}

func NewFile(rawFile *os.File, flags int) *File {
	return &File{
		rawFile:  rawFile,
		readonly: flags&FlagRead == FlagRead,
	}
}

func (this *File) Name() string {
	return this.rawFile.Name()
}

func (this *File) Fd() uintptr {
	return this.rawFile.Fd()
}

func (this *File) Raw() *os.File {
	return this.rawFile
}

func (this *File) Stat() (os.FileInfo, error) {
	return this.rawFile.Stat()
}

func (this *File) Seek(offset int64, whence int) (ret int64, err error) {
	ret, err = this.rawFile.Seek(offset, whence)
	return
}

func (this *File) Read(b []byte) (n int, err error) {
	ReaderLimiter.Ack()
	n, err = this.rawFile.Read(b)
	ReaderLimiter.Release()
	return
}

func (this *File) ReadAt(b []byte, off int64) (n int, err error) {
	ReaderLimiter.Ack()
	n, err = this.rawFile.ReadAt(b, off)
	ReaderLimiter.Release()
	return
}

func (this *File) Write(b []byte) (n int, err error) {
	WriterLimiter.Ack()
	n, err = this.rawFile.Write(b)
	WriterLimiter.Release()
	return
}

func (this *File) WriteAt(b []byte, off int64) (n int, err error) {
	WriterLimiter.Ack()
	n, err = this.rawFile.WriteAt(b, off)
	WriterLimiter.Release()
	return
}

func (this *File) Sync() (err error) {
	WriterLimiter.Ack()
	err = this.rawFile.Sync()
	WriterLimiter.Release()
	return
}

func (this *File) Truncate(size int64) (err error) {
	WriterLimiter.Ack()
	err = this.rawFile.Truncate(size)
	WriterLimiter.Release()
	return
}

func (this *File) Close() (err error) {
	if !this.readonly {
		WriterLimiter.Ack()
	}
	err = this.rawFile.Close()
	if !this.readonly {
		WriterLimiter.Release()
	}
	return
}
