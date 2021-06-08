// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package caches

import "errors"

// 常用的几个错误
var (
	ErrNotFound      = errors.New("cache not found")
	ErrFileIsWriting = errors.New("the file is writing")
	ErrInvalidRange  = errors.New("invalid range")
)

// CapacityError 容量错误
// 独立出来是为了可以在有些场合下可以忽略，防止产生没必要的错误提示数量太多
type CapacityError struct {
	err string
}

func NewCapacityError(err string) error {
	return &CapacityError{err: err}
}

func (this *CapacityError) Error() string {
	return this.err
}

// CanIgnoreErr 检查错误是否可以忽略
func CanIgnoreErr(err error) bool {
	if err == nil {
		return true
	}
	if err == ErrFileIsWriting {
		return true
	}
	_, ok := err.(*CapacityError)
	if ok {
		return true
	}
	return false
}
