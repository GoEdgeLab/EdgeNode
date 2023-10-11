// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package requests

import (
	"bytes"
	"io"
	"net"
	"net/http"
)

type TestRequest struct {
	req      *http.Request
	BodyData []byte
}

func NewTestRequest(raw *http.Request) *TestRequest {
	return &TestRequest{
		req: raw,
	}
}

func (this *TestRequest) WAFSetCacheBody(bodyData []byte) {
	this.BodyData = bodyData
}

func (this *TestRequest) WAFGetCacheBody() []byte {
	return this.BodyData
}

func (this *TestRequest) WAFRaw() *http.Request {
	return this.req
}

func (this *TestRequest) WAFRemoteAddr() string {
	return this.req.RemoteAddr
}

func (this *TestRequest) WAFRemoteIP() string {
	host, _, err := net.SplitHostPort(this.req.RemoteAddr)
	if err != nil {
		return this.req.RemoteAddr
	} else {
		return host
	}
}

func (this *TestRequest) WAFReadBody(max int64) (data []byte, err error) {
	if this.req.ContentLength > 0 {
		data, err = io.ReadAll(io.LimitReader(this.req.Body, max))
	}
	return
}

func (this *TestRequest) WAFRestoreBody(data []byte) {
	if len(data) > 0 {
		rawReader := bytes.NewBuffer(data)
		buf := make([]byte, 1024)
		_, _ = io.CopyBuffer(rawReader, this.req.Body, buf)
		this.req.Body = io.NopCloser(rawReader)
	}
}

func (this *TestRequest) WAFServerId() int64 {
	return 0
}

// WAFClose 关闭当前请求所在的连接
func (this *TestRequest) WAFClose() {
}

func (this *TestRequest) Format(s string) string {
	return s
}

func (this *TestRequest) WAFOnAction(action any) bool {
	return true
}

func (this *TestRequest) WAFFingerprint() []byte {
	return nil
}

func (this *TestRequest) DisableAccessLog() {

}

func (this *TestRequest) ProcessResponseHeaders(headers http.Header, status int) {

}

func (this *TestRequest) WAFMaxRequestSize() int64 {
	return 1 << 20
}
