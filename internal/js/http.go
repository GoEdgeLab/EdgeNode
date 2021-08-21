// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package js

type HTTP struct {
	r RequestInterface

	req  *Request
	resp *Response

	onRequest func(req *Request, resp *Response)
}

func NewHTTP(r RequestInterface) *HTTP {
	return &HTTP{
		req:  NewRequest(r),
		resp: NewResponse(r),
	}
}

func (this *HTTP) OnRequest(callback func(req *Request, resp *Response)) {
	// TODO 考虑是否支持多个callback
	this.onRequest = callback
}

func (this *HTTP) OnData(callback func(req *Request, resp *Response)) {
	// TODO
}

func (this *HTTP) OnResponse(callback func(req *Request, resp *Response)) {
	// TODO
}

func (this *HTTP) TriggerRequest() {
	this.onRequest(this.req, this.resp)
}
