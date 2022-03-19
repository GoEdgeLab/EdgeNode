// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package compressions

import "io"

type Reader interface {
	Read(p []byte) (n int, err error)
	Reset(reader io.Reader) error
	RawClose() error
	Close() error

	SetPool(pool *ReaderPool)
	ResetFinish()
}
