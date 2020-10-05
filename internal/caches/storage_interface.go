package caches

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
)

// 缓存存储接口
type StorageInterface interface {
	// 初始化
	Init() error

	// 读取缓存
	Read(key string, readerBuf []byte, callback func(data []byte, size int64, expiredAt int64, isEOF bool)) error

	// 打开缓存写入器等待写入
	Open(key string, expiredAt int64) (Writer, error)

	// 删除某个键值对应的缓存
	Delete(key string) error

	// 统计缓存
	Stat() (*Stat, error)

	// 清除所有缓存
	CleanAll() error

	// 批量删除缓存
	Purge(keys []string) error

	// 停止缓存策略
	Stop()

	// 获取当前存储的Policy
	Policy() *serverconfigs.HTTPCachePolicy

	// 将缓存添加到列表
	AddToList(item *Item)
}
