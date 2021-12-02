package utils

import (
	"time"
)

var unixTime = time.Now().Unix()
var unixTimeMilli = time.Now().UnixMilli()

func init() {
	ticker := time.NewTicker(200 * time.Millisecond)
	go func() {
		for range ticker.C {
			unixTime = time.Now().Unix()
			unixTimeMilli = time.Now().UnixMilli()
		}
	}()
}

// UnixTime 最快获取时间戳的方式，通常用在不需要特别精确时间戳的场景
func UnixTime() int64 {
	return unixTime
}

// UnixTimeMilli 获取时间戳，精确到毫秒
func UnixTimeMilli() int64 {
	return unixTimeMilli
}
