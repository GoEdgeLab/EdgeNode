package caches

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
)

// StorageInterface 缓存存储接口
type StorageInterface interface {
	// Init 初始化
	Init() error

	// OpenReader 读取缓存
	OpenReader(key string, useStale bool, isPartial bool) (reader Reader, err error)

	// OpenWriter 打开缓存写入器等待写入
	// size 和 maxSize 可能为-1
	OpenWriter(key string, expiresAt int64, status int, headerSize int, bodySize int64, maxSize int64, isPartial bool) (Writer, error)

	// OpenFlushWriter 打开从其他媒介直接刷入的写入器
	OpenFlushWriter(key string, expiresAt int64, status int, headerSize int, bodySize int64) (Writer, error)

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
	// urlType 值为file|dir
	Purge(keys []string, urlType string) error

	// Stop 停止缓存策略
	Stop()

	// Policy 获取当前存储的Policy
	Policy() *serverconfigs.HTTPCachePolicy

	// UpdatePolicy 修改策略
	UpdatePolicy(newPolicy *serverconfigs.HTTPCachePolicy)

	// CanUpdatePolicy 检查策略是否可以更新
	CanUpdatePolicy(newPolicy *serverconfigs.HTTPCachePolicy) bool

	// AddToList 将缓存添加到列表
	AddToList(item *Item)

	// IgnoreKey 忽略某个Key，即不缓存某个Key
	IgnoreKey(key string, maxSize int64)

	// CanSendfile 是否支持Sendfile
	CanSendfile() bool
}
