package utils

import (
	"time"
)

var unixTime = time.Now().Unix()
var unixTimerIsReady = false

func init() {
	ticker := time.NewTicker(500 * time.Millisecond)
	go func() {
		for range ticker.C {
			unixTimerIsReady = true
			unixTime = time.Now().Unix()
		}
	}()
}

// 最快获取时间戳的方式，通常用在不需要特别精确时间戳的场景
func UnixTime() int64 {
	if unixTimerIsReady {
		return unixTime
	}
	return time.Now().Unix()
}
