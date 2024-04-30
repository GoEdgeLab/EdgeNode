// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package fsutils

import (
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"runtime"
	"time"
)

var maxThreads = runtime.NumCPU()
var WriterLimiter = NewLimiter(maxThreads)
var ReaderLimiter = NewLimiter(maxThreads)

type Limiter struct {
	threads chan struct{}
	timers  chan *time.Timer
}

func NewLimiter(threads int) *Limiter {
	if threads < 4 {
		threads = 4
	}
	if threads > 32 {
		threads = 32
	}

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
	const timeoutDuration = 1 * time.Second

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
	select {
	case this.threads <- struct{}{}:
	default:
		remotelogs.Error("FS_LIMITER", "Limiter Ack()/Release() should appeared as a pair")
	}
}

func (this *Limiter) FreeThreads() int {
	return len(this.threads)
}
