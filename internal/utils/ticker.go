package utils

import (
	"github.com/TeaOSLab/EdgeNode/internal/zero"
	"sync"
	"time"
)

// Ticker 类似于time.Ticker，但能够真正地停止
type Ticker struct {
	raw  *time.Ticker
	done chan zero.Zero
	once sync.Once

	C <-chan time.Time
}

// NewTicker 创建新Ticker
func NewTicker(duration time.Duration) *Ticker {
	raw := time.NewTicker(duration)
	return &Ticker{
		raw:  raw,
		C:    raw.C,
		done: make(chan zero.Zero, 1),
	}
}

// Next 查找下一个Tick
func (this *Ticker) Next() bool {
	select {
	case <-this.raw.C:
		return true
	case <-this.done:
		return false
	}
}

// Stop 停止
func (this *Ticker) Stop() {
	this.once.Do(func() {
		this.raw.Stop()
		this.done <- zero.New()
	})
}
