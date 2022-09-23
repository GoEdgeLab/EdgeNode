package nodes

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/shared"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/iwind/TeaGo/Tea"
	"github.com/iwind/TeaGo/logs"
	"net/http"
	"os"
	"regexp"
)

var urlPrefixRegexp = regexp.MustCompile("^(?i)(http|https|ftp)://")

// 请求特殊页面
func (this *HTTPRequest) doPage(status int) (shouldStop bool) {
	if len(this.web.Pages) == 0 {
		return false
	}

	for _, page := range this.web.Pages {
		if page.Match(status) {
			if len(page.BodyType) == 0 || page.BodyType == shared.BodyTypeURL {
				if urlPrefixRegexp.MatchString(page.URL) {
					var newStatus = page.NewStatus
					if newStatus <= 0 {
						newStatus = status
					}
					this.doURL(http.MethodGet, page.URL, "", newStatus, true)
					return true
				} else {
					file := Tea.Root + Tea.DS + page.URL
					fp, err := os.Open(file)
					if err != nil {
						logs.Error(err)
						msg := "404 page not found: '" + page.URL + "'"

						this.writer.WriteHeader(http.StatusNotFound)
						_, err := this.writer.Write([]byte(msg))
						if err != nil {
							logs.Error(err)
						}
						return true
					}

					stat, err := fp.Stat()
					if err != nil {
						logs.Error(err)
						msg := "404 could not read page content: '" + page.URL + "'"

						this.writer.WriteHeader(http.StatusNotFound)
						_, err := this.writer.Write([]byte(msg))
						if err != nil {
							logs.Error(err)
						}
						return true
					}

					// 修改状态码
					if page.NewStatus > 0 {
						// 自定义响应Headers
						this.processResponseHeaders(this.writer.Header(), page.NewStatus)
						this.writer.Prepare(nil, stat.Size(), page.NewStatus, true)
						this.writer.WriteHeader(page.NewStatus)
					} else {
						this.processResponseHeaders(this.writer.Header(), status)
						this.writer.Prepare(nil, stat.Size(), status, true)
						this.writer.WriteHeader(status)
					}
					buf := utils.BytePool1k.Get()
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
					err = fp.Close()
					if err != nil {
						logs.Error(err)
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
					this.processResponseHeaders(this.writer.Header(), page.NewStatus)
					this.writer.Prepare(nil, int64(len(content)), page.NewStatus, true)
					this.writer.WriteHeader(page.NewStatus)
				} else {
					this.processResponseHeaders(this.writer.Header(), status)
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
