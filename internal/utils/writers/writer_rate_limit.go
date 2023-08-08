// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package writers

import (
	"context"
	"github.com/iwind/TeaGo/types"
	"io"
	"time"
)

// RateLimitWriter 限速写入
type RateLimitWriter struct {
	rawWriter io.WriteCloser
	ctx       context.Context

	rateBytes int

	written int
	before  time.Time
}

func NewRateLimitWriter(ctx context.Context, rawWriter io.WriteCloser, rateBytes int64) io.WriteCloser {
	return &RateLimitWriter{
		rawWriter: rawWriter,
		ctx:       ctx,
		rateBytes: types.Int(rateBytes),
		before:    time.Now(),
	}
}

func (this *RateLimitWriter) Write(p []byte) (n int, err error) {
	if this.rateBytes <= 0 {
		return this.write(p)
	}

	var size = len(p)
	if size == 0 {
		return 0, nil
	}

	if size <= this.rateBytes {
		return this.write(p)
	}

	for {
		size = len(p)

		var limit = this.rateBytes
		if limit > size {
			limit = size
		}
		n1, wErr := this.write(p[:limit])
		n += n1
		if wErr != nil {
			return n, wErr
		}

		if size > limit {
			p = p[limit:]
		} else {
			break
		}
	}

	return
}

func (this *RateLimitWriter) Close() error {
	return this.rawWriter.Close()
}

func (this *RateLimitWriter) write(p []byte) (n int, err error) {
	n, err = this.rawWriter.Write(p)

	if err == nil {
		select {
		case <-this.ctx.Done():
			err = io.EOF
			return
		default:
		}

		this.written += n

		if this.written >= this.rateBytes {
			var duration = 1*time.Second - time.Since(this.before)
			if duration > 0 {
				time.Sleep(duration)
			}
			this.before = time.Now()
			this.written = 0
		}
	}

	return
}
