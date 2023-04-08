// Copyright 2023 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package fasttime

import (
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/iwind/TeaGo/types"
	timeutil "github.com/iwind/TeaGo/utils/time"
	"time"
)

var sharedFastTime = NewFastTime()

func init() {
	if !teaconst.IsMain {
		return
	}

	var ticker = time.NewTicker(200 * time.Millisecond)
	goman.New(func() {
		for range ticker.C {
			sharedFastTime = NewFastTime()
		}
	})
}

func Now() *FastTime {
	return sharedFastTime
}

type FastTime struct {
	rawTime             time.Time
	unixTime            int64
	unixTimeMilli       int64
	unixTimeMilliString string
	ymd                 string
	round5Hi            string
}

func NewFastTime() *FastTime {
	var rawTime = time.Now()

	return &FastTime{
		rawTime:             rawTime,
		unixTime:            rawTime.Unix(),
		unixTimeMilli:       rawTime.UnixMilli(),
		unixTimeMilliString: types.String(rawTime.UnixMilli()),
		ymd:                 timeutil.Format("Ymd", rawTime),
		round5Hi:            timeutil.FormatTime("Hi", rawTime.Unix()/300*300),
	}
}

// Unix 最快获取时间戳的方式，通常用在不需要特别精确时间戳的场景
func (this *FastTime) Unix() int64 {
	return this.unixTime
}

// UnixFloor 取整
func (this *FastTime) UnixFloor(seconds int) int64 {
	return this.unixTime / int64(seconds) * int64(seconds)
}

// UnixCell 取整并加1
func (this *FastTime) UnixCell(seconds int) int64 {
	return this.unixTime/int64(seconds)*int64(seconds) + int64(seconds)
}

// UnixNextMinute 获取下一分钟开始的时间戳
func (this *FastTime) UnixNextMinute() int64 {
	return this.UnixCell(60)
}

// UnixMilli 获取时间戳，精确到毫秒
func (this *FastTime) UnixMilli() int64 {
	return this.unixTimeMilli
}

func (this *FastTime) UnixMilliString() (int64, string) {
	return this.unixTimeMilli, this.unixTimeMilliString
}

func (this *FastTime) Ymd() string {
	return this.ymd
}

func (this *FastTime) Round5Hi() string {
	return this.round5Hi
}

func (this *FastTime) Format(layout string) string {
	return timeutil.Format(layout, this.rawTime)
}
