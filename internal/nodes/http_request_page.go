package nodes

import (
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
			if urlPrefixRegexp.MatchString(page.URL) {
				this.doURL(http.MethodGet, page.URL, "", page.NewStatus)
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

				// 修改状态码
				if page.NewStatus > 0 {
					// 自定义响应Headers
					this.processResponseHeaders(page.NewStatus)
					this.writer.WriteHeader(page.NewStatus)
				} else {
					this.processResponseHeaders(status)
					this.writer.WriteHeader(status)
				}
				buf := bytePool1k.Get()
				_, err = utils.CopyWithFilter(this.writer, fp, buf, func(p []byte) []byte {
					return []byte(this.Format(string(p)))
				})
				bytePool1k.Put(buf)
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
		}
	}
	return false
}
