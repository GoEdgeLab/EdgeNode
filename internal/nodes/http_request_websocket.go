package nodes

import (
	"errors"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/shared"
	"github.com/iwind/TeaGo/logs"
	"io"
	"net/http"
	"net/url"
)

// 处理Websocket请求
func (this *HTTPRequest) doWebsocket() {
	if this.web.WebsocketRef == nil || !this.web.WebsocketRef.IsOn || this.web.Websocket == nil || !this.web.Websocket.IsOn {
		this.writer.WriteHeader(http.StatusForbidden)
		return
	}

	// TODO 实现handshakeTimeout

	// 校验来源
	requestOrigin := this.RawReq.Header.Get("Origin")
	if len(requestOrigin) > 0 {
		u, err := url.Parse(requestOrigin)
		if err == nil {
			if !this.web.Websocket.MatchOrigin(u.Host) {
				this.writer.WriteHeader(http.StatusForbidden)
				return
			}
		}
	}

	requestCall := shared.NewRequestCall()
	origin := this.reverseProxy.NextOrigin(requestCall)
	if origin == nil {
		err := errors.New(this.requestPath() + ": no available backends for websocket")
		logs.Error(err)
		this.addError(err)
		this.write500()
		return
	}

	// 处理Header
	this.processRequestHeaders(this.RawReq.Header)
	this.fixRequestHeader(this.RawReq.Header) // 处理 Websocket -> WebSocket

	// 设置指定的来源域
	if !this.web.Websocket.RequestSameOrigin && len(this.web.Websocket.RequestOrigin) > 0 {
		newRequestOrigin := this.web.Websocket.RequestOrigin
		if this.web.Websocket.RequestOriginHasVariables() {
			newRequestOrigin = this.Format(newRequestOrigin)
		}
		this.RawReq.Header.Set("Origin", newRequestOrigin)
	}

	// TODO 增加N次错误重试，重试的时候需要尝试不同的源站
	originConn, err := OriginConnect(origin)
	if err != nil {
		logs.Error(err)
		this.addError(err)
		this.write500()
		return
	}
	defer func() {
		_ = originConn.Close()
	}()

	err = this.RawReq.Write(originConn)
	if err != nil {
		logs.Error(err)
		this.addError(err)
		this.write500()
		return
	}

	clientConn, _, err := this.writer.Hijack()
	if err != nil {
		logs.Error(err)
		this.addError(err)
		this.write500()
		return
	}
	defer func() {
		_ = clientConn.Close()
	}()

	go func() {
		_, _ = io.Copy(clientConn, originConn)
		_ = clientConn.Close()
		_ = originConn.Close()
	}()
	_, _ = io.Copy(originConn, clientConn)
}
