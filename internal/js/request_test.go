// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package js_test

import (
	"bytes"
	"github.com/TeaOSLab/EdgeNode/internal/js"
	"github.com/iwind/TeaGo/logs"
	"io/ioutil"
	"net/http"
	"testing"
)

type testRequest struct {
	rawRequest  *http.Request
	rawResponse *testResponse
}

func (this *testRequest) JSRequest() *http.Request {
	if this.rawRequest != nil {
		return this.rawRequest
	}
	req, _ := http.NewRequest(http.MethodGet, "https://iwind:123456@goedge.cn/docs?name=Libai&age=20", nil)
	req.Header.Set("Server", "edgejs/1.0")
	req.Header.Set("Content-Type", "application/json")
	req.Body = ioutil.NopCloser(bytes.NewReader([]byte("123456")))
	this.rawRequest = req
	return req
}

func (this *testRequest) JSWriter() http.ResponseWriter {
	if this.rawResponse != nil {
		return this.rawResponse
	}
	this.rawResponse = &testResponse{}
	return this.rawResponse
}

func (this *testRequest) JSStop() {

}

func (this *testRequest) JSLog(msg ...interface{}) {
	logs.Println(msg...)
}

type testResponse struct {
	statusCode int
	header     http.Header
}

func (this *testResponse) Header() http.Header {
	if this.header == nil {
		this.header = http.Header{}
	}
	return this.header
}

func (this *testResponse) Write(p []byte) (int, error) {
	return len(p), nil
}

func (this *testResponse) WriteHeader(statusCode int) {
	this.statusCode = statusCode
}

func TestRequest(t *testing.T) {
	vm := js.NewVM()
	vm.SetRequest(&testRequest{})

	// 事件监听
	_, err := vm.RunString(`
	http.onRequest(function (req, resp) {
		console.log(req.proto())

		let url = req.url()
		console.log(url, "port:", url.port(), "args:", url.args())
		console.log("username:", url.username(), "password:", url.password())
		console.log("uri:", url.uri(), "path:", url.path())

		req.addHeader("Server", "1.0")

		
		resp.write("this is response")
		console.log(resp)

		console.log(req.body()) 
	})
`)
	if err != nil {
		t.Fatal(err)
	}

	// 触发事件
	_, err = vm.RunString(`http.triggerRequest()`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRequest_Header(t *testing.T) {
	var req = js.NewRequest(&testRequest{})
	logs.PrintAsJSON(req.Header(), t)

	req.AddHeader("Content-Length", "10")
	req.AddHeader("Vary", "1.0")
	req.AddHeader("Vary", "2.0")
	logs.PrintAsJSON(req.Header(), t)

	req.SetHeader("Vary", "3.0")
	logs.PrintAsJSON(req.Header(), t)
}

func TestRequest_Body(t *testing.T) {
	var req = js.NewRequest(&testRequest{})
	t.Log(string(req.Body()))
	t.Log(string(req.Body()))
}

func TestRequest_CopyBody(t *testing.T) {
	var req = js.NewRequest(&testRequest{})
	t.Log(string(req.CopyBody()))
	t.Log(string(req.CopyBody()))
}
