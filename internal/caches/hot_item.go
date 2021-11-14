// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package caches

type HotItem struct {
	Key       string
	ExpiresAt int64
	Hits      uint32
	Status int
}
