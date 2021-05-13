package caches

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
)

// StorageInterface 缓存存储接口
type StorageInterface interface {
	// Init 初始化
	Init() error

	// OpenReader 读取缓存
	OpenReader(key string) (Reader, error)

	// OpenWriter 打开缓存写入器等待写入
	OpenWriter(key string, expiredAt int64, status int) (Writer, error)

	// Delete 删除某个键值对应的缓存
	Delete(key string) error

	// Stat 统计缓存
	Stat() (*Stat, error)

	// TotalDiskSize 消耗的磁盘尺寸
	TotalDiskSize() int64

	// TotalMemorySize 内存尺寸
	TotalMemorySize() int64

	// CleanAll 清除所有缓存
	CleanAll() error

	// Purge 批量删除缓存
	Purge(keys []string, urlType string) error

	// Stop 停止缓存策略
	Stop()

	// Policy 获取当前存储的Policy
	Policy() *serverconfigs.HTTPCachePolicy

	// AddToList 将缓存添加到列表
	AddToList(item *Item)
}
