// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package nodes

import (
	"bufio"
	"github.com/iwind/TeaGo/types"
	"net"
	"net/http"
	"time"
)

// HTTPRateWriter 限速写入
type HTTPRateWriter struct {
	parentWriter http.ResponseWriter

	rateBytes int
	lastBytes int
	timeCost  time.Duration
}

func NewHTTPRateWriter(writer http.ResponseWriter, rateBytes int64) http.ResponseWriter {
	return &HTTPRateWriter{
		parentWriter: writer,
		rateBytes:    types.Int(rateBytes),
	}
}

func (this *HTTPRateWriter) Header() http.Header {
	return this.parentWriter.Header()
}

func (this *HTTPRateWriter) Write(data []byte) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}

	var left = this.rateBytes - this.lastBytes

	if left <= 0 {
		if this.timeCost > 0 && this.timeCost < 1*time.Second {
			time.Sleep(1*time.Second - this.timeCost)
		}

		this.lastBytes = 0
		this.timeCost = 0
		return this.Write(data)
	}

	var n = len(data)

	// n <= left
	if n <= left {
		this.lastBytes += n

		var before = time.Now()
		defer func() {
			this.timeCost += time.Since(before)
		}()
		return this.parentWriter.Write(data)
	}

	// n > left
	var before = time.Now()
	result, err := this.parentWriter.Write(data[:left])
	this.timeCost += time.Since(before)

	if err != nil {
		return result, err
	}
	this.lastBytes += left

	return this.Write(data[left:])
}

func (this *HTTPRateWriter) WriteHeader(statusCode int) {
	this.parentWriter.WriteHeader(statusCode)
}

// Hijack Hijack
func (this *HTTPRateWriter) Hijack() (conn net.Conn, buf *bufio.ReadWriter, err error) {
	if this.parentWriter == nil {
		return
	}
	hijack, ok := this.parentWriter.(http.Hijacker)
	if ok {
		return hijack.Hijack()
	}
	return
}
