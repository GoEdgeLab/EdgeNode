// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package nodes

import (
	"bufio"
	"net"
	"net/http"
)

// EmptyResponseWriter 空的响应Writer
type EmptyResponseWriter struct {
	header       http.Header
	parentWriter http.ResponseWriter

	statusCode int
}

func NewEmptyResponseWriter(parentWriter http.ResponseWriter) *EmptyResponseWriter {
	return &EmptyResponseWriter{
		header:       http.Header{},
		parentWriter: parentWriter,
	}
}

func (this *EmptyResponseWriter) Header() http.Header {
	return this.header
}

func (this *EmptyResponseWriter) Write(data []byte) (int, error) {
	if this.statusCode > 300 && this.parentWriter != nil {
		return this.parentWriter.Write(data)
	}
	return 0, nil
}

func (this *EmptyResponseWriter) WriteHeader(statusCode int) {
	this.statusCode = statusCode

	if this.statusCode > 300 && this.parentWriter != nil {
		var parentHeader = this.parentWriter.Header()
		for k, v := range this.header {
			parentHeader[k] = v
		}
		this.parentWriter.WriteHeader(this.statusCode)
	}
}

func (this *EmptyResponseWriter) StatusCode() int {
	return this.statusCode
}

// Hijack Hijack
func (this *EmptyResponseWriter) Hijack() (conn net.Conn, buf *bufio.ReadWriter, err error) {
	if this.parentWriter == nil {
		return
	}
	hijack, ok := this.parentWriter.(http.Hijacker)
	if ok {
		return hijack.Hijack()
	}
	return
}
