package nodes

import (
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/iwind/TeaGo/logs"
	"io"
	"net/http"
	"time"
)

// 请求某个URL
func (this *HTTPRequest) doURL(method string, url string, host string, statusCode int, supportVariables bool) {
	req, err := http.NewRequest(method, url, this.RawReq.Body)
	if err != nil {
		logs.Error(err)
		return
	}

	// 修改Host
	if len(host) > 0 {
		req.Host = this.Format(host)
	}

	// 添加当前Header
	req.Header = this.RawReq.Header

	// 代理头部
	this.setForwardHeaders(req.Header)

	// 自定义请求Header
	this.processRequestHeaders(req.Header)

	var client = utils.SharedHttpClient(60 * time.Second)
	resp, err := client.Do(req)
	if err != nil {
		remotelogs.Error("HTTP_REQUEST_URL", req.URL.String()+": "+err.Error())
		this.write50x(err, http.StatusInternalServerError, "Failed to read url", "读取URL失败", false)
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// Header
	if statusCode <= 0 {
		this.ProcessResponseHeaders(this.writer.Header(), resp.StatusCode)
	} else {
		this.ProcessResponseHeaders(this.writer.Header(), statusCode)
	}

	if supportVariables {
		resp.Header.Del("Content-Length")
	}
	this.writer.AddHeaders(resp.Header)
	if statusCode <= 0 {
		this.writer.Prepare(resp, resp.ContentLength, resp.StatusCode, true)
	} else {
		this.writer.Prepare(resp, resp.ContentLength, statusCode, true)
	}

	// 设置响应代码
	if statusCode <= 0 {
		this.writer.WriteHeader(resp.StatusCode)
	} else {
		this.writer.WriteHeader(statusCode)
	}

	// 输出内容
	var pool = this.bytePool(resp.ContentLength)
	var buf = pool.Get()
	if supportVariables {
		_, err = utils.CopyWithFilter(this.writer, resp.Body, buf, func(p []byte) []byte {
			return []byte(this.Format(string(p)))
		})
	} else {
		_, err = io.CopyBuffer(this.writer, resp.Body, buf)
	}
	pool.Put(buf)

	if err != nil {
		if !this.canIgnore(err) {
			remotelogs.Warn("HTTP_REQUEST_URL", "write to client failed: "+err.Error())
		}
	} else {
		this.writer.SetOk()
	}
}
