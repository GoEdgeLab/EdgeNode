package utils

import (
	"github.com/TeaOSLab/EdgeNode/internal/re"
	"github.com/TeaOSLab/EdgeNode/internal/ttlcache"
	"github.com/TeaOSLab/EdgeNode/internal/utils/fasttime"
	"github.com/cespare/xxhash"
	"strconv"
)

var cache = ttlcache.NewCache[int8]()

const (
	maxCacheDataSize = 1024
)

type CacheLife = int64

const (
	CacheDisabled   CacheLife = 0
	CacheShortLife  CacheLife = 600
	CacheMiddleLife CacheLife = 1800
	CacheLongLife   CacheLife = 7200
)

// MatchStringCache 正则表达式匹配字符串，并缓存结果
func MatchStringCache(regex *re.Regexp, s string, cacheLife CacheLife) bool {
	if regex == nil {
		return false
	}

	// 如果长度超过一定数量，大概率是不能重用的
	if cacheLife <= 0 || len(s) > maxCacheDataSize {
		return regex.MatchString(s)
	}

	var hash = xxhash.Sum64String(s)
	var key = regex.IdString() + "@" + strconv.FormatUint(hash, 10)
	var item = cache.Read(key)
	if item != nil {
		return item.Value == 1
	}
	var b = regex.MatchString(s)
	if b {
		cache.Write(key, 1, fasttime.Now().Unix()+cacheLife)
	} else {
		cache.Write(key, 0, fasttime.Now().Unix()+cacheLife)
	}
	return b
}

// MatchBytesCache 正则表达式匹配字节slice，并缓存结果
func MatchBytesCache(regex *re.Regexp, byteSlice []byte, cacheLife CacheLife) bool {
	if regex == nil {
		return false
	}

	// 如果长度超过一定数量，大概率是不能重用的
	if cacheLife <= 0 || len(byteSlice) > maxCacheDataSize {
		return regex.Match(byteSlice)
	}

	var hash = xxhash.Sum64(byteSlice)
	var key = regex.IdString() + "@" + strconv.FormatUint(hash, 10)
	var item = cache.Read(key)
	if item != nil {
		return item.Value == 1
	}
	if item != nil {
		return item.Value == 1
	}
	var b = regex.Match(byteSlice)
	if b {
		cache.Write(key, 1, fasttime.Now().Unix()+cacheLife)
	} else {
		cache.Write(key, 0, fasttime.Now().Unix()+cacheLife)
	}
	return b
}
