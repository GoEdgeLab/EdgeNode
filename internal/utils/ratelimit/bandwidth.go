// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package ratelimit

import (
	"context"
	"github.com/TeaOSLab/EdgeNode/internal/utils/fasttime"
	"sync/atomic"
	"time"
)

// Bandwidth lossy bandwidth limiter
type Bandwidth struct {
	totalBytes int64

	currentTimestamp int64
	currentBytes     int64
}

// NewBandwidth create new bandwidth limiter
func NewBandwidth(totalBytes int64) *Bandwidth {
	return &Bandwidth{totalBytes: totalBytes}
}

// Ack acquire next chance to send data
func (this *Bandwidth) Ack(ctx context.Context, newBytes int) {
	if newBytes <= 0 {
		return
	}
	if this.totalBytes <= 0 {
		return
	}

	var timestamp = fasttime.Now().Unix()
	if this.currentTimestamp != 0 && this.currentTimestamp != timestamp {
		this.currentTimestamp = timestamp
		this.currentBytes = int64(newBytes)

		// 第一次发送直接放行，不需要判断
		return
	}

	if this.currentTimestamp == 0 {
		this.currentTimestamp = timestamp
	}
	if atomic.AddInt64(&this.currentBytes, int64(newBytes)) <= this.totalBytes {
		return
	}

	var timeout = time.NewTimer(1 * time.Second)
	if ctx != nil {
		select {
		case <-timeout.C:
		case <-ctx.Done():
		}
	} else {
		select {
		case <-timeout.C:
		}
	}
}
