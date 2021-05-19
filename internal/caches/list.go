// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package caches

type ListInterface interface {
	Init() error

	Reset() error

	Add(hash string, item *Item) error

	Exist(hash string) (bool, error)

	// FindKeysWithPrefix 根据前缀进行查找
	FindKeysWithPrefix(prefix string) (keys []string, err error)

	Remove(hash string) error

	Purge(count int, callback func(hash string) error) error

	CleanAll() error

	Stat(check func(hash string) bool) (*Stat, error)

	// Count 总数量
	Count() (int64, error)

	// OnAdd 添加事件
	OnAdd(f func(item *Item))

	// OnRemove 删除事件
	OnRemove(f func(item *Item))
}
