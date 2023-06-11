package nodes

import (
	"bufio"
	"bytes"
	"errors"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"io"
	"net/http"
	"net/url"
)

// WebsocketResponseReader Websocket响应Reader
type WebsocketResponseReader struct {
	rawReader io.Reader
	buf       []byte
}

func NewWebsocketResponseReader(rawReader io.Reader) *WebsocketResponseReader {
	return &WebsocketResponseReader{
		rawReader: rawReader,
	}
}

func (this *WebsocketResponseReader) Read(p []byte) (n int, err error) {
	n, err = this.rawReader.Read(p)
	if n > 0 {
		if len(this.buf) == 0 {
			this.buf = make([]byte, n)
			copy(this.buf, p[:n])
		} else {
			this.buf = append(this.buf, p[:n]...)
		}
	}
	return
}

// 处理Websocket请求
func (this *HTTPRequest) doWebsocket(requestHost string, isLastRetry bool) (shouldRetry bool) {
	// 设置不缓存
	this.web.Cache = nil

	if this.web.WebsocketRef == nil || !this.web.WebsocketRef.IsOn || this.web.Websocket == nil || !this.web.Websocket.IsOn {
		this.writer.WriteHeader(http.StatusForbidden)
		this.addError(errors.New("websocket have not been enabled yet"))
		return
	}

	// TODO 实现handshakeTimeout

	// 校验来源
	var requestOrigin = this.RawReq.Header.Get("Origin")
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
		var newRequestOrigin = this.web.Websocket.RequestOrigin
		if this.web.Websocket.RequestOriginHasVariables() {
			newRequestOrigin = this.Format(newRequestOrigin)
		}
		this.RawReq.Header.Set("Origin", newRequestOrigin)
	}

	// 获取当前连接
	var requestConn = this.RawReq.Context().Value(HTTPConnContextKey)
	if requestConn == nil {
		return
	}

	// 连接源站
	// TODO 增加N次错误重试，重试的时候需要尝试不同的源站
	originConn, _, err := OriginConnect(this.origin, this.requestServerPort(), this.RawReq.RemoteAddr, requestHost)
	if err != nil {
		if isLastRetry {
			this.write50x(err, http.StatusBadGateway, "Failed to connect origin site", "源站连接失败", false)
		}

		// 增加失败次数
		SharedOriginStateManager.Fail(this.origin, requestHost, this.reverseProxy, func() {
			this.reverseProxy.ResetScheduling()
		})

		shouldRetry = true
		return
	}

	if !this.origin.IsOk {
		SharedOriginStateManager.Success(this.origin, func() {
			this.reverseProxy.ResetScheduling()
		})
	}

	defer func() {
		_ = originConn.Close()
	}()

	err = this.RawReq.Write(originConn)
	if err != nil {
		this.write50x(err, http.StatusBadGateway, "Failed to write request to origin site", "源站请求初始化失败", false)
		return
	}

	requestClientConn, ok := requestConn.(ClientConnInterface)
	if ok {
		requestClientConn.SetIsPersistent(true)
	}

	clientConn, _, err := this.writer.Hijack()
	if err != nil || clientConn == nil {
		this.write50x(err, http.StatusInternalServerError, "Failed to get origin site connection", "获取源站连接失败", false)
		return
	}
	defer func() {
		_ = clientConn.Close()
	}()

	go func() {
		// 读取第一个响应
		var respReader = NewWebsocketResponseReader(originConn)
		resp, err := http.ReadResponse(bufio.NewReader(respReader), this.RawReq)
		if err != nil || resp == nil {
			if resp != nil && resp.Body != nil {
				_ = resp.Body.Close()
			}

			_ = clientConn.Close()
			_ = originConn.Close()
			return
		}

		this.ProcessResponseHeaders(resp.Header, resp.StatusCode)
		this.writer.statusCode = resp.StatusCode

		// 将响应写回客户端
		err = resp.Write(clientConn)
		if err != nil {
			if resp.Body != nil {
				_ = resp.Body.Close()
			}

			_ = clientConn.Close()
			_ = originConn.Close()
			return
		}

		// 剩余已经从源站读取的内容
		var headerBytes = respReader.buf
		var headerIndex = bytes.Index(headerBytes, []byte{'\r', '\n', '\r', '\n'}) // CRLF
		if headerIndex > 0 {
			var leftBytes = headerBytes[headerIndex+4:]
			if len(leftBytes) > 0 {
				_, err = clientConn.Write(leftBytes)
				if err != nil {
					if resp.Body != nil {
						_ = resp.Body.Close()
					}

					_ = clientConn.Close()
					_ = originConn.Close()
					return
				}
			}
		}

		if resp.Body != nil {
			_ = resp.Body.Close()
		}

		// 复制剩余的数据
		var buf = utils.BytePool4k.Get()
		defer utils.BytePool4k.Put(buf)
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

	return
}
