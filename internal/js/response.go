// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package js

type Response struct {
	r RequestInterface
}

func NewResponse(r RequestInterface) *Response {
	return &Response{
		r: r,
	}
}

func (this *Response) Write(s string) error {
	_, err := this.r.JSWriter().Write([]byte(s))
	return err
}

func (this *Response) Reply(status int) {
	this.SetStatus(status)
	this.r.JSStop()
}

func (this *Response) Header() map[string][]string {
	return this.r.JSWriter().Header()
}

func (this *Response) AddHeader(name string, value string) {
	this.r.JSWriter().Header()[name] = append(this.r.JSWriter().Header()[name], value)
}

func (this *Response) SetHeader(name string, value string) {
	this.r.JSWriter().Header()[name] = []string{value}
}

func (this *Response) SetStatus(statusCode int) {
	this.r.JSWriter().WriteHeader(statusCode)
}
