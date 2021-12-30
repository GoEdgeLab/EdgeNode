package nodes

import (
	"github.com/iwind/TeaGo/lists"
	"github.com/iwind/TeaGo/types"
	"net/http"
)

func (this *HTTPRequest) write404() {
	if this.doPage(http.StatusNotFound) {
		return
	}

	this.processResponseHeaders(http.StatusNotFound)
	this.writer.WriteHeader(http.StatusNotFound)
	_, _ = this.writer.Write([]byte("404 page not found: '" + this.URL() + "'" + " (Request Id: " + this.requestId + ")"))
}

func (this *HTTPRequest) writeCode(code int) {
	if this.doPage(code) {
		return
	}

	this.processResponseHeaders(code)
	this.writer.WriteHeader(code)
	_, _ = this.writer.Write([]byte(types.String(code) + " " + http.StatusText(code) + ": '" + this.URL() + "'" + " (Request Id: " + this.requestId + ")"))
}

func (this *HTTPRequest) write50x(err error, statusCode int, canTryStale bool) {
	if err != nil {
		this.addError(err)
	}

	// 尝试从缓存中恢复
	if canTryStale &&
		this.cacheCanTryStale &&
		this.web.Cache.Stale != nil &&
		this.web.Cache.Stale.IsOn &&
		(len(this.web.Cache.Stale.Status) == 0 || lists.ContainsInt(this.web.Cache.Stale.Status, statusCode)) {
		ok := this.doCacheRead(true)
		if ok {
			return
		}
	}

	// 显示自定义页面
	if this.doPage(statusCode) {
		return
	}
	this.processResponseHeaders(statusCode)
	this.writer.WriteHeader(statusCode)
	_, _ = this.writer.Write([]byte(types.String(statusCode) + " " + http.StatusText(statusCode) + " (Request Id: " + this.requestId + ")"))
}
