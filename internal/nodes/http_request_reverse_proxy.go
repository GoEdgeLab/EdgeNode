package nodes

import (
	"context"
	"errors"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/shared"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// 处理反向代理
func (this *HTTPRequest) doReverseProxy() {
	if this.reverseProxy == nil {
		return
	}

	// 对URL的处理
	stripPrefix := this.reverseProxy.StripPrefix
	requestURI := this.reverseProxy.RequestURI
	requestURIHasVariables := this.reverseProxy.RequestURIHasVariables()

	var requestHost = ""
	if this.reverseProxy.RequestHostType == serverconfigs.RequestHostTypeCustomized {
		requestHost = this.reverseProxy.RequestHost
	}
	requestHostHasVariables := this.reverseProxy.RequestHostHasVariables()

	// 源站
	requestCall := shared.NewRequestCall()
	requestCall.Request = this.RawReq
	requestCall.Formatter = this.Format
	requestCall.Domain = this.Host
	origin := this.reverseProxy.NextOrigin(requestCall)
	requestCall.CallResponseCallbacks(this.writer)
	if origin == nil {
		err := errors.New(this.requestFullURL() + ": no available origin sites for reverse proxy")
		remotelogs.ServerError(this.Server.Id, "HTTP_REQUEST_REVERSE_PROXY", err.Error(), "", nil)
		this.write50x(err, http.StatusBadGateway)
		return
	}
	this.origin = origin // 设置全局变量是为了日志等处理
	if len(origin.StripPrefix) > 0 {
		stripPrefix = origin.StripPrefix
	}
	if len(origin.RequestURI) > 0 {
		requestURI = origin.RequestURI
		requestURIHasVariables = origin.RequestURIHasVariables()
	}
	if len(origin.RequestHost) > 0 {
		requestHost = origin.RequestHost
		requestHostHasVariables = origin.RequestHostHasVariables()
	}

	// 处理Scheme
	if origin.Addr == nil {
		err := errors.New(this.requestFullURL() + ": origin '" + strconv.FormatInt(origin.Id, 10) + "' does not has a address")
		remotelogs.Error("HTTP_REQUEST_REVERSE_PROXY", err.Error())
		this.write50x(err, http.StatusBadGateway)
		return
	}
	this.RawReq.URL.Scheme = origin.Addr.Protocol.Primary().Scheme()

	// StripPrefix
	if len(stripPrefix) > 0 {
		if stripPrefix[0] != '/' {
			stripPrefix = "/" + stripPrefix
		}
		this.uri = strings.TrimPrefix(this.uri, stripPrefix)
		if len(this.uri) == 0 || this.uri[0] != '/' {
			this.uri = "/" + this.uri
		}
	}

	// RequestURI
	if len(requestURI) > 0 {
		if requestURIHasVariables {
			this.uri = this.Format(requestURI)
		} else {
			this.uri = requestURI
		}
		if len(this.uri) == 0 || this.uri[0] != '/' {
			this.uri = "/" + this.uri
		}

		// 处理RequestURI中的问号
		questionMark := strings.LastIndex(this.uri, "?")
		if questionMark > 0 {
			path := this.uri[:questionMark]
			if strings.Contains(path, "?") {
				this.uri = path + "&" + this.uri[questionMark+1:]
			}
		}

		// 去除多个/
		this.uri = utils.CleanPath(this.uri)
	}

	// 获取源站地址
	originAddr := origin.Addr.PickAddress()
	if origin.Addr.HostHasVariables() {
		originAddr = this.Format(originAddr)
	}
	this.originAddr = originAddr

	// RequestHost
	if len(requestHost) > 0 {
		if requestHostHasVariables {
			this.RawReq.Host = this.Format(requestHost)
		} else {
			this.RawReq.Host = this.reverseProxy.RequestHost
		}
		this.RawReq.URL.Host = this.RawReq.Host
	} else if this.reverseProxy.RequestHostType == serverconfigs.RequestHostTypeOrigin {
		this.RawReq.Host = originAddr
		this.RawReq.URL.Host = this.RawReq.Host
	} else {
		this.RawReq.URL.Host = this.Host
	}

	// 重组请求URL
	questionMark := strings.Index(this.uri, "?")
	if questionMark > -1 {
		this.RawReq.URL.Path = this.uri[:questionMark]
		this.RawReq.URL.RawQuery = this.uri[questionMark+1:]
	} else {
		this.RawReq.URL.Path = this.uri
		this.RawReq.URL.RawQuery = ""
	}
	this.RawReq.RequestURI = ""

	// 处理Header
	this.setForwardHeaders(this.RawReq.Header)
	this.processRequestHeaders(this.RawReq.Header)

	// 判断是否为Websocket请求
	if this.RawReq.Header.Get("Upgrade") == "websocket" {
		this.doWebsocket()
		return
	}

	// 获取请求客户端
	client, err := SharedHTTPClientPool.Client(this, origin, originAddr, this.reverseProxy.ProxyProtocol)
	if err != nil {
		remotelogs.Error("HTTP_REQUEST_REVERSE_PROXY", err.Error())
		this.write50x(err, http.StatusBadGateway)
		return
	}

	// 在HTTP/2下需要防止因为requestBody而导致Content-Length为空的问题
	if this.RawReq.ProtoMajor == 2 && this.RawReq.ContentLength == 0 {
		_ = this.RawReq.Body.Close()
		this.RawReq.Body = nil
	}

	// 开始请求
	resp, err := client.Do(this.RawReq)
	if err != nil {
		// 客户端取消请求，则不提示
		httpErr, ok := err.(*url.Error)
		if !ok {
			SharedOriginStateManager.Fail(origin, this.reverseProxy, func() {
				this.reverseProxy.ResetScheduling()
			})
			this.write50x(err, http.StatusBadGateway)
			remotelogs.Warn("HTTP_REQUEST_REVERSE_PROXY", this.RawReq.URL.String()+"': "+err.Error())
		} else if httpErr.Err != context.Canceled {
			SharedOriginStateManager.Fail(origin, this.reverseProxy, func() {
				this.reverseProxy.ResetScheduling()
			})
			if httpErr.Timeout() {
				this.write50x(err, http.StatusGatewayTimeout)
			} else if httpErr.Temporary() {
				this.write50x(err, http.StatusServiceUnavailable)
			} else {
				this.write50x(err, http.StatusBadGateway)
			}
			remotelogs.Warn("HTTP_REQUEST_REVERSE_PROXY", this.RawReq.URL.String()+"': "+err.Error())
		} else {
			// 是否为客户端方面的错误
			isClientError := false
			if ok {
				if httpErr.Err == context.Canceled {
					// 如果是服务器端主动关闭，则无需提示
					if this.isConnClosed() {
						this.disableLog = true
						return
					}

					isClientError = true
					this.addError(errors.New(httpErr.Op + " " + httpErr.URL + ": client closed the connection"))
					this.writer.WriteHeader(499) // 仿照nginx
				}
			}

			if !isClientError {
				this.write50x(err, http.StatusBadGateway)
			}
		}
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		return
	}
	if !origin.IsOk {
		SharedOriginStateManager.Success(origin, func() {
			this.reverseProxy.ResetScheduling()
		})
	}

	// WAF对出站进行检查
	if this.web.FirewallRef != nil && this.web.FirewallRef.IsOn {
		if this.doWAFResponse(resp) {
			err = resp.Body.Close()
			if err != nil {
				remotelogs.Warn("HTTP_REQUEST_REVERSE_PROXY", err.Error())
			}
			return
		}
	}

	// TODO 清除源站错误次数

	// 特殊页面
	if len(this.web.Pages) > 0 && this.doPage(resp.StatusCode) {
		err = resp.Body.Close()
		if err != nil {
			remotelogs.Warn("HTTP_REQUEST_REVERSE_PROXY", err.Error())
		}
		return
	}

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
	this.processResponseHeaders(resp.StatusCode)

	// 是否需要刷新
	shouldAutoFlush := this.reverseProxy.AutoFlush || this.RawReq.Header.Get("Accept") == "text/event-stream"

	// 准备
	delayHeaders := this.writer.Prepare(resp.ContentLength, resp.StatusCode)

	// 设置响应代码
	if !delayHeaders {
		this.writer.WriteHeader(resp.StatusCode)
	}

	// 输出到客户端
	pool := this.bytePool(resp.ContentLength)
	buf := pool.Get()
	if shouldAutoFlush {
		for {
			n, readErr := resp.Body.Read(buf)
			if n > 0 {
				_, err = this.writer.Write(buf[:n])
				this.writer.Flush()
				if err != nil {
					break
				}
			}
			if readErr != nil {
				err = readErr
				break
			}
		}
	} else {
		_, err = io.CopyBuffer(this.writer, resp.Body, buf)
	}
	pool.Put(buf)

	closeErr := resp.Body.Close()
	if closeErr != nil {
		if !this.canIgnore(closeErr) {
			remotelogs.Warn("HTTP_REQUEST_REVERSE_PROXY", closeErr.Error())
		}
	}

	if err != nil && err != io.EOF {
		if !this.canIgnore(err) {
			remotelogs.Warn("HTTP_REQUEST_REVERSE_PROXY", err.Error())
			this.addError(err)
		}
	}

	// 是否成功结束
	if err == nil && closeErr == nil {
		this.writer.SetOk()
	}
}
