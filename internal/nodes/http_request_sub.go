// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package nodes

import "net/http"

// 执行子请求
func (this *HTTPRequest) doSubRequest(writer http.ResponseWriter, rawReq *http.Request) {
	// 包装新请求对象
	req := &HTTPRequest{
		RawReq:     rawReq,
		RawWriter:  writer,
		ReqServer:  this.ReqServer,
		ReqHost:    this.ReqHost,
		ServerName: this.ServerName,
		ServerAddr: this.ServerAddr,
		IsHTTP:     this.IsHTTP,
		IsHTTPS:    this.IsHTTPS,
	}
	req.isSubRequest = true
	req.Do()
}
