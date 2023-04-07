package utils

import (
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/iwind/TeaGo/types"
	timeutil "github.com/iwind/TeaGo/utils/time"
	"time"
)

var unixTime = time.Now().Unix()
var unixTimeMilli = time.Now().UnixMilli()
var unixTimeMilliString = types.String(unixTimeMilli)
var ymd = timeutil.Format("Ymd")
var round5Hi = timeutil.FormatTime("Hi", time.Now().Unix()/300*300)

func init() {
	if !teaconst.IsMain {
		return
	}

	var ticker = time.NewTicker(200 * time.Millisecond)
	goman.New(func() {
		for range ticker.C {
			unixTime = time.Now().Unix()
			unixTimeMilli = time.Now().UnixMilli()
			unixTimeMilliString = types.String(unixTimeMilli)
			ymd = timeutil.Format("Ymd")
			round5Hi = timeutil.FormatTime("Hi", time.Now().Unix()/300*300)
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

func UnixTimeMilliString() (int64, string) {
	return unixTimeMilli, unixTimeMilliString
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

// Ymd 读取YYYYMMDD
func Ymd() string {
	return ymd
}

// Round5Hi 读取5分钟间隔时间
func Round5Hi() string {
	return round5Hi
}
