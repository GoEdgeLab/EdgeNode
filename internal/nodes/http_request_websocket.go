package nodes

import (
	"errors"
	"github.com/iwind/TeaGo/logs"
	"io"
	"net/http"
	"net/url"
)

// 处理Websocket请求
func (this *HTTPRequest) doWebsocket() {
	if this.web.WebsocketRef == nil || !this.web.WebsocketRef.IsOn || this.web.Websocket == nil || !this.web.Websocket.IsOn {
		this.writer.WriteHeader(http.StatusForbidden)
		this.addError(errors.New("websocket have not been enabled yet"))
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
				this.addError(errors.New("websocket origin '" + requestOrigin + "' not been allowed"))
				return
			}
		}
	}

	// 设置指定的来源域
	if !this.web.Websocket.RequestSameOrigin && len(this.web.Websocket.RequestOrigin) > 0 {
		newRequestOrigin := this.web.Websocket.RequestOrigin
		if this.web.Websocket.RequestOriginHasVariables() {
			newRequestOrigin = this.Format(newRequestOrigin)
		}
		this.RawReq.Header.Set("Origin", newRequestOrigin)
	}

	// TODO 增加N次错误重试，重试的时候需要尝试不同的源站
	originConn, err := OriginConnect(this.origin, this.RawReq.RemoteAddr)
	if err != nil {
		logs.Error(err)
		this.write500(err)
		return
	}
	defer func() {
		_ = originConn.Close()
	}()

	err = this.RawReq.Write(originConn)
	if err != nil {
		logs.Error(err)
		this.write500(err)
		return
	}

	clientConn, _, err := this.writer.Hijack()
	if err != nil {
		logs.Error(err)
		this.write500(err)
		return
	}
	defer func() {
		_ = clientConn.Close()
	}()

	go func() {
		buf := make([]byte, 4*1024) // TODO 使用内存池
		for {
			n, err := originConn.Read(buf)
			if n > 0 {
				this.writer.sentBodyBytes += int64(n)
				_, err = clientConn.Write(buf[:n])
				if err != nil {
					break
				}
			}
			if err != nil {
				break
			}
		}
		_ = clientConn.Close()
		_ = originConn.Close()
	}()
	_, _ = io.Copy(originConn, clientConn)
}
