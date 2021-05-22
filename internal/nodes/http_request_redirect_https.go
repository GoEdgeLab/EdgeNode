package nodes

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"net/http"
	"strconv"
	"strings"
)

func (this *HTTPRequest) doRedirectToHTTPS(redirectToHTTPSConfig *serverconfigs.HTTPRedirectToHTTPSConfig) {
	host := this.RawReq.Host

	if len(redirectToHTTPSConfig.Host) > 0 {
		if redirectToHTTPSConfig.Port > 0 && redirectToHTTPSConfig.Port != 443 {
			host = redirectToHTTPSConfig.Host + ":" + strconv.Itoa(redirectToHTTPSConfig.Port)
		} else {
			host = redirectToHTTPSConfig.Host
		}
	} else if redirectToHTTPSConfig.Port > 0 {
		lastIndex := strings.LastIndex(host, ":")
		if lastIndex > 0 {
			host = host[:lastIndex]
		}
		if redirectToHTTPSConfig.Port != 443 {
			host = host + ":" + strconv.Itoa(redirectToHTTPSConfig.Port)
		}
	} else {
		lastIndex := strings.LastIndex(host, ":")
		if lastIndex > 0 {
			host = host[:lastIndex]
		}
	}

	statusCode := http.StatusMovedPermanently
	if redirectToHTTPSConfig.Status > 0 {
		statusCode = redirectToHTTPSConfig.Status
	}

	newURL := "https://" + host + this.RawReq.RequestURI
	http.Redirect(this.writer, this.RawReq, newURL, statusCode)
}
