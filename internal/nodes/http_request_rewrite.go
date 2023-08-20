package nodes

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"net/http"
)

// 调用Rewrite
func (this *HTTPRequest) doRewrite() (shouldShop bool) {
	if this.rewriteRule == nil {
		return
	}

	// 代理
	if this.rewriteRule.Mode == serverconfigs.HTTPRewriteModeProxy {
		// 外部URL
		if this.rewriteIsExternalURL {
			host := this.ReqHost
			if len(this.rewriteRule.ProxyHost) > 0 {
				host = this.rewriteRule.ProxyHost
			}
			this.doURL(this.RawReq.Method, this.rewriteReplace, host, 0, false)
			return true
		}

		// 内部URL继续
		return false
	}

	// 跳转
	if this.rewriteRule.Mode == serverconfigs.HTTPRewriteModeRedirect {
		if this.rewriteRule.RedirectStatus > 0 {
			this.ProcessResponseHeaders(this.writer.Header(), this.rewriteRule.RedirectStatus)
			httpRedirect(this.writer, this.RawReq, this.rewriteReplace, this.rewriteRule.RedirectStatus)
		} else {
			this.ProcessResponseHeaders(this.writer.Header(), http.StatusTemporaryRedirect)
			httpRedirect(this.writer, this.RawReq, this.rewriteReplace, http.StatusTemporaryRedirect)
		}
		return true
	}

	return true
}
