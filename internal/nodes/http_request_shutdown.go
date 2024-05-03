package nodes

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/TeaOSLab/EdgeNode/internal/utils/bytepool"
	"github.com/iwind/TeaGo/Tea"
	"net/http"
	"os"
	"path"
	"strings"
)

// 调用临时关闭页面
func (this *HTTPRequest) doShutdown() {
	var shutdown = this.web.Shutdown
	if shutdown == nil {
		return
	}

	if len(shutdown.BodyType) == 0 || shutdown.BodyType == serverconfigs.HTTPPageBodyTypeURL {
		// URL
		if urlSchemeRegexp.MatchString(shutdown.URL) {
			this.doURL(http.MethodGet, shutdown.URL, "", shutdown.Status, true)
			return
		}

		// URL为空，则显示文本
		if len(shutdown.URL) == 0 {
			// 自定义响应Headers
			if shutdown.Status > 0 {
				this.ProcessResponseHeaders(this.writer.Header(), shutdown.Status)
				this.writer.WriteHeader(shutdown.Status)
			} else {
				this.ProcessResponseHeaders(this.writer.Header(), http.StatusOK)
				this.writer.WriteHeader(http.StatusOK)
			}
			_, _ = this.writer.WriteString("The site have been shutdown.")
			return
		}

		// 从本地文件中读取
		var realpath = path.Clean(shutdown.URL)
		if !strings.HasPrefix(realpath, "/pages/") && !strings.HasPrefix(realpath, "pages/") { // only files under "/pages/" can be used
			var msg = "404 page not found: '" + shutdown.URL + "'"
			this.writer.WriteHeader(http.StatusNotFound)
			_, _ = this.writer.Write([]byte(msg))
			return
		}
		var file = Tea.Root + Tea.DS + shutdown.URL
		fp, err := os.Open(file)
		if err != nil {
			var msg = "404 page not found: '" + shutdown.URL + "'"
			this.writer.WriteHeader(http.StatusNotFound)
			_, _ = this.writer.Write([]byte(msg))
			return
		}

		defer func() {
			_ = fp.Close()
		}()

		// 自定义响应Headers
		if shutdown.Status > 0 {
			this.ProcessResponseHeaders(this.writer.Header(), shutdown.Status)
			this.writer.WriteHeader(shutdown.Status)
		} else {
			this.ProcessResponseHeaders(this.writer.Header(), http.StatusOK)
			this.writer.WriteHeader(http.StatusOK)
		}
		var buf = bytepool.Pool1k.Get()
		_, err = utils.CopyWithFilter(this.writer, fp, buf.Bytes, func(p []byte) []byte {
			return []byte(this.Format(string(p)))
		})
		bytepool.Pool1k.Put(buf)
		if err != nil {
			if !this.canIgnore(err) {
				remotelogs.Warn("HTTP_REQUEST_SHUTDOWN", "write to client failed: "+err.Error())
			}
		} else {
			this.writer.SetOk()
		}
	} else if shutdown.BodyType == serverconfigs.HTTPPageBodyTypeHTML {
		// 自定义响应Headers
		if shutdown.Status > 0 {
			this.ProcessResponseHeaders(this.writer.Header(), shutdown.Status)
			this.writer.WriteHeader(shutdown.Status)
		} else {
			this.ProcessResponseHeaders(this.writer.Header(), http.StatusOK)
			this.writer.WriteHeader(http.StatusOK)
		}

		_, err := this.writer.WriteString(this.Format(shutdown.Body))
		if err != nil {
			if !this.canIgnore(err) {
				remotelogs.Warn("HTTP_REQUEST_SHUTDOWN", "write to client failed: "+err.Error())
			}
		} else {
			this.writer.SetOk()
		}
	} else if shutdown.BodyType == serverconfigs.HTTPPageBodyTypeRedirectURL {
		var newURL = shutdown.URL
		if len(newURL) == 0 {
			newURL = "/"
		}

		if shutdown.Status > 0 && httpStatusIsRedirect(shutdown.Status) {
			httpRedirect(this.writer, this.RawReq, newURL, shutdown.Status)
		} else {
			httpRedirect(this.writer, this.RawReq, newURL, http.StatusTemporaryRedirect)
		}
		this.writer.SetOk()
	}
}
