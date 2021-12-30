package nodes

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"net/http"
	"strconv"
	"strings"
)

// 主机地址快速跳转
func (this *HTTPRequest) doHostRedirect() (blocked bool) {
	var urlPath = this.RawReq.URL.Path
	if this.web.MergeSlashes {
		urlPath = utils.CleanPath(urlPath)
	}
	fullURL := this.requestScheme() + "://" + this.host + urlPath
	for _, u := range this.web.HostRedirects {
		if !u.IsOn {
			continue
		}
		if !u.MatchRequest(this.Format) {
			continue
		}
		if u.MatchPrefix { // 匹配前缀
			if strings.HasPrefix(fullURL, u.BeforeURL) {
				afterURL := u.AfterURL
				if u.KeepRequestURI {
					afterURL += this.RawReq.URL.RequestURI()
				}

				// 前后是否一致
				if fullURL == afterURL {
					return false
				}

				if u.Status <= 0 {
					this.processResponseHeaders(http.StatusTemporaryRedirect)
					http.Redirect(this.RawWriter, this.RawReq, afterURL, http.StatusTemporaryRedirect)
				} else {
					this.processResponseHeaders(u.Status)
					http.Redirect(this.RawWriter, this.RawReq, afterURL, u.Status)
				}
				return true
			}
		} else if u.MatchRegexp { // 正则匹配
			reg := u.BeforeURLRegexp()
			if reg == nil {
				continue
			}
			matches := reg.FindStringSubmatch(fullURL)
			if len(matches) == 0 {
				continue
			}
			afterURL := u.AfterURL
			for i, match := range matches {
				afterURL = strings.ReplaceAll(afterURL, "${"+strconv.Itoa(i)+"}", match)
			}

			subNames := reg.SubexpNames()
			if len(subNames) > 0 {
				for _, subName := range subNames {
					if len(subName) > 0 {
						index := reg.SubexpIndex(subName)
						if index > -1 {
							afterURL = strings.ReplaceAll(afterURL, "${"+subName+"}", matches[index])
						}
					}
				}
			}

			// 前后是否一致
			if fullURL == afterURL {
				return false
			}

			if u.Status <= 0 {
				this.processResponseHeaders(http.StatusTemporaryRedirect)
				http.Redirect(this.RawWriter, this.RawReq, afterURL, http.StatusTemporaryRedirect)
			} else {
				this.processResponseHeaders(u.Status)
				http.Redirect(this.RawWriter, this.RawReq, afterURL, u.Status)
			}
			return true
		} else { // 精准匹配
			if fullURL == u.RealBeforeURL() {
				// 前后是否一致
				if fullURL == u.AfterURL {
					return false
				}

				if u.Status <= 0 {
					this.processResponseHeaders(http.StatusTemporaryRedirect)
					http.Redirect(this.RawWriter, this.RawReq, u.AfterURL, http.StatusTemporaryRedirect)
				} else {
					this.processResponseHeaders(u.Status)
					http.Redirect(this.RawWriter, this.RawReq, u.AfterURL, u.Status)
				}
				return true
			}
		}
	}
	return
}
