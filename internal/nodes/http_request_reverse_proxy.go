package nodes

// 处理反向代理
func (this *HTTPRequest) doReverseProxy() {
	// 判断是否为Websocket请求
	if this.RawReq.Header.Get("Upgrade") == "websocket" {
		this.doWebsocket()
		return
	}

	// 普通HTTP请求
	// TODO
}

