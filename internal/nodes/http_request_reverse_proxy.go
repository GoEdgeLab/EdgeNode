package nodes

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"strings"
)

// 处理反向代理
func (this *HTTPRequest) doReverseProxy() {
	if this.reverseProxy == nil {
		return
	}

	// StripPrefix
	if len(this.reverseProxy.StripPrefix) > 0 {
		stripPrefix := this.reverseProxy.StripPrefix
		if stripPrefix[0] != '/' {
			stripPrefix = "/" + stripPrefix
		}
		this.uri = strings.TrimPrefix(this.uri, stripPrefix)
		if len(this.uri) == 0 || this.uri[0] != '/' {
			this.uri = "/" + this.uri
		}
	}

	// RequestURI
	if len(this.reverseProxy.RequestURI) > 0 {
		if this.reverseProxy.RequestURIHasVariables() {
			this.uri = this.Format(this.reverseProxy.RequestURI)
		} else {
			this.uri = this.reverseProxy.RequestURI
		}
		if len(this.uri) == 0 || this.uri[0] != '/' {
			this.uri = "/" + this.uri
		}

		// 处理RequestURI中的问号
		questionMark := strings.LastIndex(this.uri, "?")
		if questionMark > 0 {
			path := this.uri[:questionMark]
			if strings.Contains(path, "?") {
				this.uri = path + "&" + this.uri[questionMark+1:]
			}
		}

		// 去除多个/
		this.uri = utils.CleanPath(this.uri)
	}

	// 重组请求URL
	questionMark := strings.Index(this.uri, "?")
	if questionMark > -1 {
		this.RawReq.URL.Path = this.uri[:questionMark]
		this.RawReq.URL.RawQuery = this.uri[questionMark+1:]
	} else {
		this.RawReq.URL.Path = this.uri
		this.RawReq.URL.RawQuery = ""
	}

	// RequestHost
	if len(this.reverseProxy.RequestHost) > 0 {
		if this.reverseProxy.RequestHostHasVariables() {
			this.RawReq.Host = this.Format(this.reverseProxy.RequestHost)
		} else {
			this.RawReq.Host = this.reverseProxy.RequestHost
		}
		this.RawReq.URL.Host = this.RawReq.Host
	}

	// 判断是否为Websocket请求
	if this.RawReq.Header.Get("Upgrade") == "websocket" {
		this.doWebsocket()
		return
	}

	// 普通HTTP请求
	// TODO
}
