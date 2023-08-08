// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package nodes

import (
	"errors"
	"fmt"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/iwind/TeaGo/Tea"
	"github.com/iwind/TeaGo/maps"
	"github.com/iwind/TeaGo/rands"
	"github.com/iwind/TeaGo/types"
	"github.com/iwind/gofcgi/pkg/fcgi"
	"io"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
)

func (this *HTTPRequest) doFastcgi() (shouldStop bool) {
	fastcgiList := []*serverconfigs.HTTPFastcgiConfig{}
	for _, fastcgi := range this.web.FastcgiList {
		if !fastcgi.IsOn {
			continue
		}
		fastcgiList = append(fastcgiList, fastcgi)
	}
	if len(fastcgiList) == 0 {
		return false
	}
	shouldStop = true
	fastcgi := fastcgiList[rands.Int(0, len(fastcgiList)-1)]

	env := fastcgi.FilterParams()
	if !env.Has("DOCUMENT_ROOT") {
		env["DOCUMENT_ROOT"] = ""
	}

	if !env.Has("REMOTE_ADDR") {
		env["REMOTE_ADDR"] = this.requestRemoteAddr(true)
	}
	if !env.Has("QUERY_STRING") {
		u, err := url.ParseRequestURI(this.uri)
		if err == nil {
			env["QUERY_STRING"] = u.RawQuery
		} else {
			env["QUERY_STRING"] = this.RawReq.URL.RawQuery
		}
	}
	if !env.Has("SERVER_NAME") {
		env["SERVER_NAME"] = this.ReqHost
	}
	if !env.Has("REQUEST_URI") {
		env["REQUEST_URI"] = this.uri
	}
	if !env.Has("HOST") {
		env["HOST"] = this.ReqHost
	}

	if len(this.ServerAddr) > 0 {
		if !env.Has("SERVER_ADDR") {
			env["SERVER_ADDR"] = this.ServerAddr
		}
		if !env.Has("SERVER_PORT") {
			_, port, err := net.SplitHostPort(this.ServerAddr)
			if err == nil {
				env["SERVER_PORT"] = port
			}
		}
	}

	// 设置为持久化连接
	var requestConn = this.RawReq.Context().Value(HTTPConnContextKey)
	if requestConn == nil {
		return
	}
	requestClientConn, ok := requestConn.(ClientConnInterface)
	if ok {
		requestClientConn.SetIsPersistent(true)
	}

	// 连接池配置
	poolSize := fastcgi.PoolSize
	if poolSize <= 0 {
		poolSize = 32
	}

	client, err := fcgi.SharedPool(fastcgi.Network(), fastcgi.RealAddress(), uint(poolSize)).Client()
	if err != nil {
		this.write50x(err, http.StatusInternalServerError, "Failed to create Fastcgi pool", "Fastcgi池生成失败", false)
		return
	}

	// 请求相关
	if !env.Has("REQUEST_METHOD") {
		env["REQUEST_METHOD"] = this.RawReq.Method
	}
	if !env.Has("CONTENT_LENGTH") {
		env["CONTENT_LENGTH"] = fmt.Sprintf("%d", this.RawReq.ContentLength)
	}
	if !env.Has("CONTENT_TYPE") {
		env["CONTENT_TYPE"] = this.RawReq.Header.Get("Content-Type")
	}
	if !env.Has("SERVER_SOFTWARE") {
		env["SERVER_SOFTWARE"] = teaconst.ProductName + "/v" + teaconst.Version
	}

	// 处理SCRIPT_FILENAME
	scriptPath := env.GetString("SCRIPT_FILENAME")
	if len(scriptPath) > 0 && !strings.Contains(scriptPath, "/") && !strings.Contains(scriptPath, "\\") {
		env["SCRIPT_FILENAME"] = env.GetString("DOCUMENT_ROOT") + Tea.DS + scriptPath
	}
	scriptFilename := filepath.Base(this.RawReq.URL.Path)

	// PATH_INFO
	pathInfoReg := fastcgi.PathInfoRegexp()
	pathInfo := ""
	if pathInfoReg != nil {
		matches := pathInfoReg.FindStringSubmatch(this.RawReq.URL.Path)
		countMatches := len(matches)
		if countMatches == 1 {
			pathInfo = matches[0]
		} else if countMatches == 2 {
			pathInfo = matches[1]
		} else if countMatches > 2 {
			scriptFilename = matches[1]
			pathInfo = matches[2]
		}

		if !env.Has("PATH_INFO") {
			env["PATH_INFO"] = pathInfo
		}
	}

	this.addVarMapping(map[string]string{
		"fastcgi.documentRoot": env.GetString("DOCUMENT_ROOT"),
		"fastcgi.filename":     scriptFilename,
		"fastcgi.pathInfo":     pathInfo,
	})

	params := map[string]string{}
	for key, value := range env {
		params[key] = this.Format(types.String(value))
	}

	this.processRequestHeaders(this.RawReq.Header)
	for k, v := range this.RawReq.Header {
		if k == "Connection" {
			continue
		}
		for _, subV := range v {
			params["HTTP_"+strings.ToUpper(strings.Replace(k, "-", "_", -1))] = subV
		}
	}

	host, found := params["HTTP_HOST"]
	if !found || len(host) == 0 {
		params["HTTP_HOST"] = this.ReqHost
	}

	fcgiReq := fcgi.NewRequest()
	fcgiReq.SetTimeout(fastcgi.ReadTimeoutDuration())
	fcgiReq.SetParams(params)
	fcgiReq.SetBody(this.RawReq.Body, uint32(this.requestLength()))

	resp, stderr, err := client.Call(fcgiReq)
	if err != nil {
		this.write50x(err, http.StatusInternalServerError, "Failed to read Fastcgi", "读取Fastcgi失败", false)
		return
	}

	if len(stderr) > 0 {
		err := errors.New("Fastcgi Error: " + strings.TrimSpace(string(stderr)) + " script: " + maps.NewMap(params).GetString("SCRIPT_FILENAME"))
		this.write50x(err, http.StatusInternalServerError, "Failed to read Fastcgi", "读取Fastcgi失败", false)
		return
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	// 设置Charset
	// TODO 这里应该可以设置文本类型的列表，以及是否强制覆盖所有文本类型的字符集
	if this.web.Charset != nil && this.web.Charset.IsOn && len(this.web.Charset.Charset) > 0 {
		contentTypes, ok := resp.Header["Content-Type"]
		if ok && len(contentTypes) > 0 {
			contentType := contentTypes[0]
			if _, found := textMimeMap[contentType]; found {
				resp.Header["Content-Type"][0] = contentType + "; charset=" + this.web.Charset.Charset
			}
		}
	}

	// 响应Header
	this.writer.AddHeaders(resp.Header)
	this.ProcessResponseHeaders(this.writer.Header(), resp.StatusCode)

	// 准备
	this.writer.Prepare(resp, resp.ContentLength, resp.StatusCode, true)

	// 设置响应代码
	this.writer.WriteHeader(resp.StatusCode)

	// 输出到客户端
	pool := this.bytePool(resp.ContentLength)
	buf := pool.Get()
	_, err = io.CopyBuffer(this.writer, resp.Body, buf)
	pool.Put(buf)

	closeErr := resp.Body.Close()
	if closeErr != nil {
		remotelogs.Warn("HTTP_REQUEST_FASTCGI", closeErr.Error())
	}

	if err != nil && err != io.EOF {
		remotelogs.Warn("HTTP_REQUEST_FASTCGI", err.Error())
		this.addError(err)
	}

	// 是否成功结束
	if err == nil && closeErr == nil {
		this.writer.SetOk()
	}

	return
}
