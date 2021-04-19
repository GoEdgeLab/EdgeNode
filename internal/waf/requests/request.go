package requests

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
)

type Request struct {
	*http.Request
	BodyData []byte
}

func NewRequest(raw *http.Request) *Request {
	return &Request{
		Request: raw,
	}
}

func (this *Request) Raw() *http.Request {
	return this.Request
}

func (this *Request) ReadBody(max int64) (data []byte, err error) {
	if this.Request.ContentLength > 0 {
		data, err = ioutil.ReadAll(io.LimitReader(this.Request.Body, max))
	}
	return
}

func (this *Request) RestoreBody(data []byte) {
	if len(data) > 0 {
		rawReader := bytes.NewBuffer(data)
		buf := make([]byte, 1024)
		_, _ = io.CopyBuffer(rawReader, this.Request.Body, buf)
		this.Request.Body = ioutil.NopCloser(rawReader)
	}
}
