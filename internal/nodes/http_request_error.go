package nodes

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/configutils"
	"github.com/iwind/TeaGo/lists"
	"github.com/iwind/TeaGo/types"
	"net/http"
	"strings"
)

const httpStatusPageTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
	<title>${status} ${statusMessage}</title>
	<meta http-equiv="Content-Type" content="text/html; charset=utf-8"/>
	<style>
	address { line-height: 1.8; }
	</style>
</head>
<body>

<h1>${status} ${statusMessage}</h1>
<p>${message}</p>

<address>Connection: ${remoteAddr} (Client) -&gt; ${serverAddr} (Server)</address>
<address>Request ID: ${requestId}.</address>

</body>
</html>`

func (this *HTTPRequest) write404() {
	this.writeCode(http.StatusNotFound, "", "")
}

func (this *HTTPRequest) writeCode(statusCode int, enMessage string, zhMessage string) {
	if this.doPage(statusCode) {
		return
	}

	var pageContent = configutils.ParseVariables(httpStatusPageTemplate, func(varName string) (value string) {
		switch varName {
		case "status":
			return types.String(statusCode)
		case "statusMessage":
			return http.StatusText(statusCode)
		case "message":
			var acceptLanguages = this.RawReq.Header.Get("Accept-Language")
			if len(acceptLanguages) > 0 {
				var index = strings.Index(acceptLanguages, ",")
				if index > 0 {
					var firstLanguage = acceptLanguages[:index]
					if firstLanguage == "zh-CN" {
						return zhMessage
					}
				}
			}
			return enMessage
		}
		return this.Format("${" + varName + "}")
	})

	this.ProcessResponseHeaders(this.writer.Header(), statusCode)
	this.writer.WriteHeader(statusCode)

	_, _ = this.writer.Write([]byte(pageContent))
}

func (this *HTTPRequest) write50x(err error, statusCode int, enMessage string, zhMessage string, canTryStale bool) {
	if err != nil {
		this.addError(err)
	}

	// 尝试从缓存中恢复
	if canTryStale &&
		this.cacheCanTryStale &&
		this.web.Cache.Stale != nil &&
		this.web.Cache.Stale.IsOn &&
		(len(this.web.Cache.Stale.Status) == 0 || lists.ContainsInt(this.web.Cache.Stale.Status, statusCode)) {
		var ok = this.doCacheRead(true)
		if ok {
			return
		}
	}

	// 显示自定义页面
	if this.doPage(statusCode) {
		return
	}

	// 内置HTML模板
	var pageContent = configutils.ParseVariables(httpStatusPageTemplate, func(varName string) (value string) {
		switch varName {
		case "status":
			return types.String(statusCode)
		case "statusMessage":
			return http.StatusText(statusCode)
		case "requestId":
			return this.requestId
		case "message":
			var acceptLanguages = this.RawReq.Header.Get("Accept-Language")
			if len(acceptLanguages) > 0 {
				var index = strings.Index(acceptLanguages, ",")
				if index > 0 {
					var firstLanguage = acceptLanguages[:index]
					if firstLanguage == "zh-CN" {
						return "网站出了一点小问题，原因：" + zhMessage + "。"
					}
				}
			}
			return "The site is unavailable now, cause: " + enMessage + "."
		}
		return this.Format("${" + varName + "}")
	})

	this.ProcessResponseHeaders(this.writer.Header(), statusCode)
	this.writer.WriteHeader(statusCode)

	_, _ = this.writer.Write([]byte(pageContent))
}
