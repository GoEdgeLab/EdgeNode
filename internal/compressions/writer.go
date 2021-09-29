// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package compressions

type Writer interface {
	Write(p []byte) (int, error)
	Flush() error
	Close() error
	Level() int
}
