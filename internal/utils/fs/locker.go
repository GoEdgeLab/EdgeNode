// Copyright 2023 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package fsutils

import (
	"os"
	"syscall"
	"time"
)

type Locker struct {
	path string
	fp   *os.File
}

func NewLocker(path string) *Locker {
	return &Locker{
		path: path + ".lock",
	}
}

func (this *Locker) TryLock() (ok bool, err error) {
	if this.fp == nil {
		fp, err := os.OpenFile(this.path, os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			return false, err
		}
		this.fp = fp
	}
	return this.tryLock()
}

func (this *Locker) Lock() error {
	if this.fp == nil {
		fp, err := os.OpenFile(this.path, os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			return err
		}
		this.fp = fp
	}

	for {
		b, err := this.tryLock()
		if err != nil {
			_ = this.fp.Close()
			return err
		}
		if b {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (this *Locker) Release() error {
	err := this.fp.Close()
	if err != nil {
		return err
	}
	this.fp = nil
	return nil
}

func (this *Locker) tryLock() (ok bool, err error) {
	err = syscall.Flock(int(this.fp.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err == nil {
		return true, nil
	}

	errno, isErrNo := err.(syscall.Errno)
	if !isErrNo {
		return
	}

	if !errno.Temporary() {
		return
	}

	err = nil // 不提示错误

	return
}
