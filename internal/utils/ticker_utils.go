package utils

import (
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"time"
)

// Every 定时运行某个函数
func Every(duration time.Duration, f func(ticker *Ticker)) *Ticker {
	ticker := NewTicker(duration)
	goman.New(func() {
		for ticker.Next() {
			f(ticker)
		}
	})

	return ticker
}
