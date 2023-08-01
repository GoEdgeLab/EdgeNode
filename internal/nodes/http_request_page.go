package nodes

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/shared"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/iwind/TeaGo/Tea"
	"net/http"
	"os"
	"path"
	"strings"
)

const defaultPageContentType = "text/html; charset=utf-8"

// 请求特殊页面
func (this *HTTPRequest) doPage(status int) (shouldStop bool) {
	if len(this.web.Pages) == 0 {
		// 集群自定义页面
		if this.nodeConfig != nil && this.ReqServer != nil {
			var httpPagesPolicy = this.nodeConfig.FindHTTPPagesPolicyWithClusterId(this.ReqServer.ClusterId)
			if httpPagesPolicy != nil && httpPagesPolicy.IsOn && len(httpPagesPolicy.Pages) > 0 {
				return this.doPageLookup(httpPagesPolicy.Pages, status)
			}
		}

		return false
	}

	// 查找当前网站自定义页面
	shouldStop = this.doPageLookup(this.web.Pages, status)
	if shouldStop {
		return
	}

	// 集群自定义页面
	if this.nodeConfig != nil && this.ReqServer != nil {
		var httpPagesPolicy = this.nodeConfig.FindHTTPPagesPolicyWithClusterId(this.ReqServer.ClusterId)
		if httpPagesPolicy != nil && httpPagesPolicy.IsOn && len(httpPagesPolicy.Pages) > 0 {
			return this.doPageLookup(httpPagesPolicy.Pages, status)
		}
	}

	return
}

func (this *HTTPRequest) doPageLookup(pages []*serverconfigs.HTTPPageConfig, status int) (shouldStop bool) {
	for _, page := range pages {
		if page.Match(status) {
			if len(page.BodyType) == 0 || page.BodyType == shared.BodyTypeURL {
				if urlSchemeRegexp.MatchString(page.URL) {
					var newStatus = page.NewStatus
					if newStatus <= 0 {
						newStatus = status
					}
					this.doURL(http.MethodGet, page.URL, "", newStatus, true)
					return true
				} else {
					var realpath = path.Clean(page.URL)
					if !strings.HasPrefix(realpath, "/pages/") && !strings.HasPrefix(realpath, "pages/") { // only files under "/pages/" can be used
						var msg = "404 page not found: '" + page.URL + "'"
						this.writer.Header().Set("Content-Type", defaultPageContentType)
						this.writer.WriteHeader(http.StatusNotFound)
						_, _ = this.writer.Write([]byte(msg))
						return true
					}
					var file = Tea.Root + Tea.DS + realpath
					fp, err := os.Open(file)
					if err != nil {
						var msg = "404 page not found: '" + page.URL + "'"
						this.writer.Header().Set("Content-Type", defaultPageContentType)
						this.writer.WriteHeader(http.StatusNotFound)
						_, _ = this.writer.Write([]byte(msg))
						return true
					}
					defer func() {
						_ = fp.Close()
					}()

					stat, err := fp.Stat()
					if err != nil {
						var msg = "404 could not read page content: '" + page.URL + "'"
						this.writer.Header().Set("Content-Type", defaultPageContentType)
						this.writer.WriteHeader(http.StatusNotFound)
						_, _ = this.writer.Write([]byte(msg))
						return true
					}

					// 修改状态码
					if page.NewStatus > 0 {
						// 自定义响应Headers
						this.writer.Header().Set("Content-Type", defaultPageContentType)
						this.ProcessResponseHeaders(this.writer.Header(), page.NewStatus)
						this.writer.Prepare(nil, stat.Size(), page.NewStatus, true)
						this.writer.WriteHeader(page.NewStatus)
					} else {
						this.writer.Header().Set("Content-Type", defaultPageContentType)
						this.ProcessResponseHeaders(this.writer.Header(), status)
						this.writer.Prepare(nil, stat.Size(), status, true)
						this.writer.WriteHeader(status)
					}
					var buf = utils.BytePool1k.Get()
					_, err = utils.CopyWithFilter(this.writer, fp, buf, func(p []byte) []byte {
						return []byte(this.Format(string(p)))
					})
					utils.BytePool1k.Put(buf)
					if err != nil {
						if !this.canIgnore(err) {
							remotelogs.Warn("HTTP_REQUEST_PAGE", "write to client failed: "+err.Error())
						}
					} else {
						this.writer.SetOk()
					}
				}

				return true
			} else if page.BodyType == shared.BodyTypeHTML {
				// 这里需要实现设置Status，因为在Format()中可以获取${status}等变量
				if page.NewStatus > 0 {
					this.writer.statusCode = page.NewStatus
				} else {
					this.writer.statusCode = status
				}
				var content = this.Format(page.Body)

				// 修改状态码
				if page.NewStatus > 0 {
					// 自定义响应Headers
					this.writer.Header().Set("Content-Type", defaultPageContentType)
					this.ProcessResponseHeaders(this.writer.Header(), page.NewStatus)
					this.writer.Prepare(nil, int64(len(content)), page.NewStatus, true)
					this.writer.WriteHeader(page.NewStatus)
				} else {
					this.writer.Header().Set("Content-Type", defaultPageContentType)
					this.ProcessResponseHeaders(this.writer.Header(), status)
					this.writer.Prepare(nil, int64(len(content)), status, true)
					this.writer.WriteHeader(status)
				}

				_, err := this.writer.WriteString(content)
				if err != nil {
					if !this.canIgnore(err) {
						remotelogs.Warn("HTTP_REQUEST_PAGE", "write to client failed: "+err.Error())
					}
				} else {
					this.writer.SetOk()
				}
				return true
			}
		}
	}
	return false
}
