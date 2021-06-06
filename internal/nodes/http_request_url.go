package nodes

import (
	"errors"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/iwind/TeaGo/logs"
	"io"
	"net/http"
	"time"
)

// 请求某个URL
func (this *HTTPRequest) doURL(method string, url string, host string, statusCode int) {
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
		logs.Error(errors.New(req.URL.String() + ": " + err.Error()))
		this.write500(err)
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// Header
	if statusCode <= 0 {
		this.processResponseHeaders(resp.StatusCode)
	} else {
		this.processResponseHeaders(statusCode)
	}

	this.writer.AddHeaders(resp.Header)
	if statusCode <= 0 {
		this.writer.Prepare(resp.ContentLength, resp.StatusCode)
	} else {
		this.writer.Prepare(resp.ContentLength, statusCode)
	}

	// 设置响应代码
	if statusCode <= 0 {
		this.writer.WriteHeader(resp.StatusCode)
	} else {
		this.writer.WriteHeader(statusCode)
	}

	// 输出内容
	pool := this.bytePool(resp.ContentLength)
	buf := pool.Get()
	_, err = io.CopyBuffer(this.writer, resp.Body, buf)
	pool.Put(buf)

	if err != nil {
		if !this.canIgnore(err) {
			remotelogs.Warn("HTTP_REQUEST_URL", "write to client failed: "+err.Error())
		}
	} else {
		this.writer.SetOk()
	}
}
