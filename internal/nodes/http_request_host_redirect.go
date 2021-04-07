package nodes

import (
	"net/http"
	"strings"
)

// 主机地址快速跳转
func (this *HTTPRequest) doHostRedirect() (blocked bool) {
	fullURL := this.requestScheme() + "://" + this.Host + this.RawReq.URL.Path
	for _, u := range this.web.HostRedirects {
		if !u.IsOn {
			continue
		}
		if u.MatchPrefix {
			if strings.HasPrefix(fullURL, u.BeforeURL) {
				afterURL := u.AfterURL
				if u.KeepRequestURI {
					afterURL += this.RawReq.URL.RequestURI()
				}
				if u.Status <= 0 {
					http.Redirect(this.RawWriter, this.RawReq, afterURL, http.StatusTemporaryRedirect)
				} else {
					http.Redirect(this.RawWriter, this.RawReq, afterURL, u.Status)
				}
				return true
			}
		} else {
			if fullURL == u.RealBeforeURL() {
				if u.Status <= 0 {
					http.Redirect(this.RawWriter, this.RawReq, u.AfterURL, http.StatusTemporaryRedirect)
				} else {
					http.Redirect(this.RawWriter, this.RawReq, u.AfterURL, u.Status)
				}
				return true
			}
		}
	}
	return
}
