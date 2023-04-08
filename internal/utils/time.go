package utils

import (
	"time"
)

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
