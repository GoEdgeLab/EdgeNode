package nodes

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/configutils"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/iwind/TeaGo/types"
	"net"
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
	for _, u := range this.web.HostRedirects {
		if !u.IsOn {
			continue
		}
		if !u.MatchRequest(this.Format) {
			continue
		}

		var status = u.Status
		if status <= 0 {
			if searchEngineRegex.MatchString(this.RawReq.UserAgent()) {
				status = http.StatusMovedPermanently
			} else {
				status = http.StatusTemporaryRedirect
			}
		}

		var fullURL string
		if u.BeforeHasQuery() {
			fullURL = this.URL()
		} else {
			fullURL = this.requestScheme() + "://" + this.ReqHost + urlPath
		}

		if len(u.Type) == 0 || u.Type == serverconfigs.HTTPHostRedirectTypeURL {
			if u.MatchPrefix { // 匹配前缀
				if strings.HasPrefix(fullURL, u.BeforeURL) {
					var afterURL = u.AfterURL
					if u.KeepRequestURI {
						afterURL += this.RawReq.URL.RequestURI()
					}

					// 前后是否一致
					if fullURL == afterURL {
						return false
					}

					this.ProcessResponseHeaders(this.writer.Header(), status)
					httpRedirect(this.writer, this.RawReq, afterURL, status)
					return true
				}
			} else if u.MatchRegexp { // 正则匹配
				var reg = u.BeforeURLRegexp()
				if reg == nil {
					continue
				}
				var matches = reg.FindStringSubmatch(fullURL)
				if len(matches) == 0 {
					continue
				}
				var afterURL = u.AfterURL
				for i, match := range matches {
					afterURL = strings.ReplaceAll(afterURL, "${"+strconv.Itoa(i)+"}", match)
				}

				var subNames = reg.SubexpNames()
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

				if u.KeepArgs {
					var qIndex = strings.Index(this.uri, "?")
					if qIndex >= 0 {
						afterURL += this.uri[qIndex:]
					}
				}

				this.ProcessResponseHeaders(this.writer.Header(), status)
				httpRedirect(this.writer, this.RawReq, afterURL, status)
				return true
			} else { // 精准匹配
				if fullURL == u.RealBeforeURL() {
					// 前后是否一致
					if fullURL == u.AfterURL {
						return false
					}

					var afterURL = u.AfterURL
					if u.KeepArgs {
						var qIndex = strings.Index(this.uri, "?")
						if qIndex >= 0 {
							var afterQIndex = strings.Index(u.AfterURL, "?")
							if afterQIndex >= 0 {
								afterURL = u.AfterURL[:afterQIndex] + this.uri[qIndex:]
							} else {
								afterURL += this.uri[qIndex:]
							}
						}
					}

					this.ProcessResponseHeaders(this.writer.Header(), status)
					httpRedirect(this.writer, this.RawReq, afterURL, status)
					return true
				}
			}
		} else if u.Type == serverconfigs.HTTPHostRedirectTypeDomain {
			if len(u.DomainAfter) == 0 {
				continue
			}

			var reqHost = this.ReqHost

			// 忽略跳转前端口
			if u.DomainBeforeIgnorePorts {
				h, _, err := net.SplitHostPort(reqHost)
				if err == nil && len(h) > 0 {
					reqHost = h
				}
			}

			var scheme = u.DomainAfterScheme
			if len(scheme) == 0 {
				scheme = this.requestScheme()
			}
			if u.DomainsAll || configutils.MatchDomains(u.DomainsBefore, reqHost) {
				var afterURL = scheme + "://" + u.DomainAfter + urlPath
				if fullURL == afterURL {
					// 终止匹配
					return false
				}

				// 如果跳转前后域名一致，则终止
				if u.DomainAfter == reqHost {
					return false
				}

				this.ProcessResponseHeaders(this.writer.Header(), status)

				// 参数
				var qIndex = strings.Index(this.uri, "?")
				if qIndex >= 0 {
					afterURL += this.uri[qIndex:]
				}

				httpRedirect(this.writer, this.RawReq, afterURL, status)
				return true
			}
		} else if u.Type == serverconfigs.HTTPHostRedirectTypePort {
			if u.PortAfter <= 0 {
				continue
			}

			var scheme = u.PortAfterScheme
			if len(scheme) == 0 {
				scheme = this.requestScheme()
			}

			reqHost, reqPort, _ := net.SplitHostPort(this.ReqHost)
			if len(reqHost) == 0 {
				reqHost = this.ReqHost
			}
			if len(reqPort) == 0 {
				switch this.requestScheme() {
				case "http":
					reqPort = "80"
				case "https":
					reqPort = "443"
				}
			}

			// 如果跳转前后端口一致，则终止
			if reqPort == types.String(u.PortAfter) {
				return false
			}

			var containsPort bool
			if u.PortsAll {
				containsPort = true
			} else {
				containsPort = u.ContainsPort(types.Int(reqPort))
			}
			if containsPort {
				var newReqHost = reqHost
				if !((scheme == "http" && u.PortAfter == 80) || (scheme == "https" && u.PortAfter == 443)) {
					newReqHost += ":" + types.String(u.PortAfter)
				}
				var afterURL = scheme + "://" + newReqHost + urlPath
				if fullURL == afterURL {
					// 终止匹配
					return false
				}

				this.ProcessResponseHeaders(this.writer.Header(), status)
				httpRedirect(this.writer, this.RawReq, afterURL, status)
				return true
			}
		}
	}
	return
}
