// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package js

import (
	"bytes"
	"io/ioutil"
	"net"
)

type Request struct {
	r RequestInterface
}

func NewRequest(r RequestInterface) *Request {
	return &Request{
		r: r,
	}
}

func (this *Request) Proto() string {
	return this.r.JSRequest().Proto
}

func (this *Request) Method() string {
	return this.r.JSRequest().Method
}

func (this *Request) Header() map[string][]string {
	return this.r.JSRequest().Header
}

func (this *Request) AddHeader(name string, value string) {
	this.r.JSRequest().Header[name] = append(this.r.JSRequest().Header[name], value)
}

func (this *Request) SetHeader(name string, value string) {
	this.r.JSRequest().Header[name] = []string{value}
}

func (this *Request) RemoteAddr() string {
	var remoteAddr = this.r.JSRequest().RemoteAddr
	host, _, err := net.SplitHostPort(remoteAddr)
	if err == nil {
		return host
	}
	return remoteAddr
}

func (this *Request) Url() *URL {
	return NewURL(this.r.JSRequest().URL)
}

func (this *Request) ContentLength() int64 {
	return this.r.JSRequest().ContentLength
}

func (this *Request) Body() []byte {
	var bodyReader = this.r.JSRequest().Body
	if bodyReader == nil {
		return []byte{}
	}
	data, err := ioutil.ReadAll(bodyReader)
	if err != nil {
		this.r.JSLog("read body failed: " + err.Error())
	}
	return data
}

func (this *Request) CopyBody() []byte {
	var bodyReader = this.r.JSRequest().Body
	if bodyReader == nil {
		return []byte{}
	}

	data, err := ioutil.ReadAll(bodyReader)
	if err != nil {
		this.r.JSLog("read body failed: " + err.Error())
	}
	this.r.JSRequest().Body = ioutil.NopCloser(bytes.NewReader(data))
	return data
}
