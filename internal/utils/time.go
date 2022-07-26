package utils

import (
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"time"
)

var unixTime = time.Now().Unix()
var unixTimeMilli = time.Now().UnixMilli()

func init() {
	var ticker = time.NewTicker(200 * time.Millisecond)
	goman.New(func() {
		for range ticker.C {
			unixTime = time.Now().Unix()
			unixTimeMilli = time.Now().UnixMilli()
		}
	})
}

// UnixTime 最快获取时间戳的方式，通常用在不需要特别精确时间戳的场景
func UnixTime() int64 {
	return unixTime
}

// FloorUnixTime 取整
func FloorUnixTime(seconds int) int64 {
	return UnixTime() / int64(seconds) * int64(seconds)
}

// CeilUnixTime 取整并加1
func CeilUnixTime(seconds int) int64 {
	return UnixTime()/int64(seconds)*int64(seconds) + int64(seconds)
}

// NextMinuteUnixTime 获取下一分钟开始的时间戳
func NextMinuteUnixTime() int64 {
	return CeilUnixTime(60)
}

// UnixTimeMilli 获取时间戳，精确到毫秒
func UnixTimeMilli() int64 {
	return unixTimeMilli
}

// GMTUnixTime 计算GMT时间戳
func GMTUnixTime(timestamp int64) int64 {
	_, offset := time.Now().Zone()
	return timestamp - int64(offset)
}

// GMTTime 计算GMT时间
func GMTTime(t time.Time) time.Time {
	_, offset := time.Now().Zone()
	return t.Add(-time.Duration(offset) * time.Second)
}
