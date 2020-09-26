package nodes

import (
	"github.com/iwind/TeaGo/Tea"
	"github.com/iwind/TeaGo/logs"
	"io"
	"net/http"
	"os"
)

// 调用临时关闭页面
func (this *HTTPRequest) doShutdown() {
	shutdown := this.web.Shutdown
	if shutdown == nil {
		return
	}

	if urlPrefixRegexp.MatchString(shutdown.URL) { // URL
		this.doURL(http.MethodGet, shutdown.URL, "", shutdown.Status)
		return
	}

	// URL为空，则显示文本 TODO 未来可以自定义文本
	if len(shutdown.URL) == 0 {
		// 自定义响应Headers
		if shutdown.Status > 0 {
			this.processResponseHeaders(shutdown.Status)
			this.writer.WriteHeader(shutdown.Status)
		} else {
			this.processResponseHeaders(http.StatusOK)
			this.writer.WriteHeader(http.StatusOK)
		}
		_, err := this.writer.WriteString("The site have been shutdown.")
		if err != nil {
			logs.Error(err)
		}

		return
	}

	// 从本地文件中读取
	// TODO 支持从数据库中读取文件
	file := Tea.Root + Tea.DS + shutdown.URL
	fp, err := os.Open(file)
	if err != nil {
		logs.Error(err)
		msg := "404 page not found: '" + shutdown.URL + "'"

		this.writer.WriteHeader(http.StatusNotFound)
		_, err = this.writer.Write([]byte(msg))
		if err != nil {
			logs.Error(err)
		}
		return
	}

	// 自定义响应Headers
	if shutdown.Status > 0 {
		this.processResponseHeaders(shutdown.Status)
		this.writer.WriteHeader(shutdown.Status)
	} else {
		this.processResponseHeaders(http.StatusOK)
		this.writer.WriteHeader(http.StatusOK)
	}
	buf := bytePool1k.Get()
	_, err = io.CopyBuffer(this.writer, fp, buf)
	bytePool1k.Put(buf)
	err = fp.Close()
	if err != nil {
		logs.Error(err)
	}
}
