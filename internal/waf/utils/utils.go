package utils

import (
	"fmt"
	"github.com/TeaOSLab/EdgeNode/internal/grids"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/dchest/siphash"
	"regexp"
	"strconv"
)

var grid = grids.NewGrid(32, grids.NewLimitCountOpt(1000_0000))

// 正则表达式匹配字符串，并缓存结果
func MatchStringCache(regex *regexp.Regexp, s string) bool {
	// 如果长度超过4096，大概率是不能重用的
	if len(s) > 4096 {
		return regex.MatchString(s)
	}

	hash := siphash.Hash(0, 0, utils.UnsafeStringToBytes(s))
	key := []byte(fmt.Sprintf("%p_", regex) + strconv.FormatUint(hash, 10))
	item := grid.Read(key)
	if item != nil {
		return item.ValueInt64 == 1
	}
	b := regex.MatchString(s)
	if b {
		grid.WriteInt64(key, 1, 1800)
	} else {
		grid.WriteInt64(key, 0, 1800)
	}
	return b
}

// 正则表达式匹配字节slice，并缓存结果
func MatchBytesCache(regex *regexp.Regexp, byteSlice []byte) bool {
	// 如果长度超过4096，大概率是不能重用的
	if len(byteSlice) > 4096 {
		return regex.Match(byteSlice)
	}

	hash := siphash.Hash(0, 0, byteSlice)
	key := []byte(fmt.Sprintf("%p_", regex) + strconv.FormatUint(hash, 10))
	item := grid.Read(key)
	if item != nil {
		return item.ValueInt64 == 1
	}
	b := regex.Match(byteSlice)
	if b {
		grid.WriteInt64(key, 1, 1800)
	} else {
		grid.WriteInt64(key, 0, 1800)
	}
	return b
}
