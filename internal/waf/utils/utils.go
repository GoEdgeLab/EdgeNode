package utils

import (
	"fmt"
	"github.com/TeaOSLab/EdgeNode/internal/re"
	"github.com/TeaOSLab/EdgeNode/internal/ttlcache"
	"github.com/cespare/xxhash"
	"github.com/iwind/TeaGo/types"
	"strconv"
	"time"
)

var cache = ttlcache.NewCache()

// MatchStringCache 正则表达式匹配字符串，并缓存结果
func MatchStringCache(regex *re.Regexp, s string) bool {
	// 如果长度超过4096，大概率是不能重用的
	if len(s) > 4096 {
		return regex.MatchString(s)
	}

	hash := xxhash.Sum64String(s)
	key := fmt.Sprintf("%p_", regex) + strconv.FormatUint(hash, 10)
	item := cache.Read(key)
	if item != nil {
		return types.Int8(item.Value) == 1
	}
	b := regex.MatchString(s)
	if b {
		cache.Write(key, 1, time.Now().Unix()+1800)
	} else {
		cache.Write(key, 0, time.Now().Unix()+1800)
	}
	return b
}

// MatchBytesCache 正则表达式匹配字节slice，并缓存结果
func MatchBytesCache(regex *re.Regexp, byteSlice []byte) bool {
	// 如果长度超过4096，大概率是不能重用的
	if len(byteSlice) > 4096 {
		return regex.Match(byteSlice)
	}

	hash := xxhash.Sum64(byteSlice)
	key := fmt.Sprintf("%p_", regex) + strconv.FormatUint(hash, 10)
	item := cache.Read(key)
	if item != nil {
		return types.Int8(item.Value) == 1
	}
	if item != nil {
		return types.Int8(item.Value) == 1
	}
	b := regex.Match(byteSlice)
	if b {
		cache.Write(key, 1, time.Now().Unix()+1800)
	} else {
		cache.Write(key, 0, time.Now().Unix()+1800)
	}
	return b
}
