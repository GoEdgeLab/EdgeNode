package nodes

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"net/http"
	"strconv"
	"strings"
)

func (this *HTTPRequest) doRedirectToHTTPS(redirectToHTTPSConfig *serverconfigs.HTTPRedirectToHTTPSConfig) (shouldBreak bool) {
	var host = this.RawReq.Host

	// 检查域名是否匹配
	if !redirectToHTTPSConfig.MatchDomain(host) {
		return false
	}

	if len(redirectToHTTPSConfig.Host) > 0 {
		if redirectToHTTPSConfig.Port > 0 && redirectToHTTPSConfig.Port != 443 {
			host = redirectToHTTPSConfig.Host + ":" + strconv.Itoa(redirectToHTTPSConfig.Port)
		} else {
			host = redirectToHTTPSConfig.Host
		}
	} else if redirectToHTTPSConfig.Port > 0 {
		var lastIndex = strings.LastIndex(host, ":")
		if lastIndex > 0 {
			host = host[:lastIndex]
		}
		if redirectToHTTPSConfig.Port != 443 {
			host = host + ":" + strconv.Itoa(redirectToHTTPSConfig.Port)
		}
	} else {
		var lastIndex = strings.LastIndex(host, ":")
		if lastIndex > 0 {
			host = host[:lastIndex]
		}
	}

	var statusCode = http.StatusMovedPermanently
	if redirectToHTTPSConfig.Status > 0 {
		statusCode = redirectToHTTPSConfig.Status
	}

	var newURL = "https://" + host + this.RawReq.RequestURI
	this.ProcessResponseHeaders(this.writer.Header(), statusCode)
	httpRedirect(this.writer, this.RawReq, newURL, statusCode)

	return true
}
