// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package caches

type ListInterface interface {
	// Init 初始化
	Init() error

	// Reset 重置数据
	Reset() error

	// Add 添加内容
	Add(hash string, item *Item) error

	// Exist 检查内容是否存在
	Exist(hash string) (bool, error)

	// CleanPrefix 清除某个前缀的缓存
	CleanPrefix(prefix string) error

	// CleanMatchKey 清除通配符匹配的Key
	CleanMatchKey(key string) error

	// CleanMatchPrefix 清除通配符匹配的前缀
	CleanMatchPrefix(prefix string) error

	// Remove 删除内容
	Remove(hash string) error

	// Purge 清理过期数据
	Purge(count int, callback func(hash string) error) (int, error)

	// PurgeLFU 清理LFU数据
	PurgeLFU(count int, callback func(hash string) error) error

	// CleanAll 清除所有缓存
	CleanAll() error

	// Stat 统计
	Stat(check func(hash string) bool) (*Stat, error)

	// Count 总数量
	Count() (int64, error)

	// OnAdd 添加事件
	OnAdd(f func(item *Item))

	// OnRemove 删除事件
	OnRemove(f func(item *Item))

	// Close 关闭
	Close() error

	// IncreaseHit 增加点击量
	IncreaseHit(hash string) error
}
