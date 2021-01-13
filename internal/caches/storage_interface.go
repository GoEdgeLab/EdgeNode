package caches

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
)

// 缓存存储接口
type StorageInterface interface {
	// 初始化
	Init() error

	// 读取缓存
	OpenReader(key string) (Reader, error)

	// 打开缓存写入器等待写入
	OpenWriter(key string, expiredAt int64, status int) (Writer, error)

	// 删除某个键值对应的缓存
	Delete(key string) error

	// 统计缓存
	Stat() (*Stat, error)

	// 清除所有缓存
	CleanAll() error

	// 批量删除缓存
	Purge(keys []string, urlType string) error

	// 停止缓存策略
	Stop()

	// 获取当前存储的Policy
	Policy() *serverconfigs.HTTPCachePolicy

	// 将缓存添加到列表
	AddToList(item *Item)
}
