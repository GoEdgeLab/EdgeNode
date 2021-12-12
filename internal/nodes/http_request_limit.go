// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package nodes

import "net/http"

func (this *HTTPRequest) doRequestLimit() (shouldStop bool) {
	// 检查请求Body尺寸
	// TODO 处理分片提交的内容
	if this.web.RequestLimit.MaxBodyBytes() > 0 &&
		this.RawReq.ContentLength > this.web.RequestLimit.MaxBodyBytes() {
		this.writeCode(http.StatusRequestEntityTooLarge)
		return true
	}

	// 设置连接相关参数
	if this.web.RequestLimit.MaxConns > 0 || this.web.RequestLimit.MaxConnsPerIP > 0 {
		requestConn := this.RawReq.Context().Value(HTTPConnContextKey)
		if requestConn != nil {
			clientConn, ok := requestConn.(ClientConnInterface)
			if ok && !clientConn.IsBound() {
				if !clientConn.Bind(this.Server.Id, this.requestRemoteAddr(true), this.web.RequestLimit.MaxConns, this.web.RequestLimit.MaxConnsPerIP) {
					this.writeCode(http.StatusTooManyRequests)
					this.closeConn()
					return true
				}
			}
		}
	}

	return false
}
