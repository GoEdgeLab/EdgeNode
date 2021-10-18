// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package compressions

type Reader interface {
	Read(p []byte) (n int, err error)
	Close() error
}
