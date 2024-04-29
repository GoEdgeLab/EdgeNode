// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package fsutils

import (
	"runtime"
	"time"
)

var WriterLimiter = NewLimiter(runtime.NumCPU())
var ReaderLimiter = NewLimiter(runtime.NumCPU())

type Limiter struct {
	threads chan struct{}
	timers  chan *time.Timer
}

func NewLimiter(threads int) *Limiter {
	var threadsChan = make(chan struct{}, threads)
	for i := 0; i < threads; i++ {
		threadsChan <- struct{}{}
	}

	return &Limiter{
		threads: threadsChan,
		timers:  make(chan *time.Timer, 2048),
	}
}

func (this *Limiter) Ack() {
	<-this.threads
}

func (this *Limiter) TryAck() bool {
	const timeoutDuration = 500 * time.Millisecond

	var timeout *time.Timer
	select {
	case timeout = <-this.timers:
		timeout.Reset(timeoutDuration)
	default:
		timeout = time.NewTimer(timeoutDuration)
	}

	defer func() {
		timeout.Stop()

		select {
		case this.timers <- timeout:
		default:
		}
	}()

	select {
	case <-this.threads:
		return true
	case <-timeout.C:
		return false
	}
}

func (this *Limiter) Release() {
	this.threads <- struct{}{}
}
