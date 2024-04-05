// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package caches

import "errors"

// 常用的几个错误
var (
	ErrNotFound                = errors.New("cache not found")
	ErrFileIsWriting           = errors.New("the cache file is updating")
	ErrInvalidRange            = errors.New("invalid range")
	ErrEntityTooLarge          = errors.New("entity too large")
	ErrWritingUnavailable      = errors.New("writing unavailable")
	ErrWritingQueueFull        = errors.New("writing queue full")
	ErrServerIsBusy            = errors.New("server is busy")
	ErrUnexpectedContentLength = errors.New("unexpected content length")
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
	if errors.Is(err, ErrFileIsWriting) ||
		errors.Is(err, ErrEntityTooLarge) ||
		errors.Is(err, ErrWritingUnavailable) ||
		errors.Is(err, ErrWritingQueueFull) ||
		errors.Is(err, ErrServerIsBusy) {
		return true
	}

	var capacityErr *CapacityError
	return errors.As(err, &capacityErr)
}

func IsCapacityError(err error) bool {
	if err == nil {
		return false
	}

	var capacityErr *CapacityError
	return errors.As(err, &capacityErr)
}
