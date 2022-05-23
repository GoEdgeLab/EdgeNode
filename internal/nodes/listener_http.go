package nodes

import (
	"context"
	"crypto/tls"
	"github.com/TeaOSLab/EdgeCommon/pkg/nodeutils"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/zero"
	"github.com/iwind/TeaGo/Tea"
	"golang.org/x/net/http2"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var httpErrorLogger = log.New(io.Discard, "", 0)
var metricNewConnMap = map[string]zero.Zero{} // remoteAddr => bool
var metricNewConnMapLocker = &sync.Mutex{}

type contextKey struct {
	key string
}

var HTTPConnContextKey = &contextKey{key: "http-conn"}

type HTTPListener struct {
	BaseListener

	Listener net.Listener

	addr       string
	isHTTP     bool
	isHTTPS    bool
	httpServer *http.Server
}

func (this *HTTPListener) Serve() error {
	this.addr = this.Group.Addr()
	this.isHTTP = this.Group.IsHTTP()
	this.isHTTPS = this.Group.IsHTTPS()

	this.httpServer = &http.Server{
		Addr:              this.addr,
		Handler:           this,
		ReadTimeout:       1 * time.Hour,    // TODO 改成可以配置
		ReadHeaderTimeout: 3 * time.Second,  // TODO 改成可以配置
		WriteTimeout:      1 * time.Hour,    // TODO 改成可以配置
		IdleTimeout:       75 * time.Second, // TODO 改成可以配置
		ConnState: func(conn net.Conn, state http.ConnState) {
			switch state {
			case http.StateNew:
				atomic.AddInt64(&this.countActiveConnections, 1)

				// 为指标存储连接信息
				if sharedNodeConfig.HasHTTPConnectionMetrics() {
					metricNewConnMapLocker.Lock()
					metricNewConnMap[conn.RemoteAddr().String()] = zero.New()
					metricNewConnMapLocker.Unlock()
				}
			case http.StateActive, http.StateIdle, http.StateHijacked:
				// Nothing to do
			case http.StateClosed:
				atomic.AddInt64(&this.countActiveConnections, -1)

				// 移除指标存储连接信息
				// 因为中途配置可能有改变，所以暂时不添加条件
				metricNewConnMapLocker.Lock()
				delete(metricNewConnMap, conn.RemoteAddr().String())
				metricNewConnMapLocker.Unlock()
			}
		},
		ConnContext: func(ctx context.Context, conn net.Conn) context.Context {
			tlsConn, ok := conn.(*tls.Conn)

			if ok {
				conn = NewClientTLSConn(tlsConn)
			}

			return context.WithValue(ctx, HTTPConnContextKey, conn)
		},
	}

	if !Tea.IsTesting() {
		this.httpServer.ErrorLog = httpErrorLogger
	}

	this.httpServer.SetKeepAlivesEnabled(true)

	// HTTP协议
	if this.isHTTP {
		err := this.httpServer.Serve(this.Listener)
		if err != nil && err != http.ErrServerClosed {
			return err
		}
	}

	// HTTPS协议
	if this.isHTTPS {
		this.httpServer.TLSConfig = this.buildTLSConfig()

		// support http/2
		err := http2.ConfigureServer(this.httpServer, nil)
		if err != nil {
			remotelogs.Error("HTTP_LISTENER", "configure http2 error: "+err.Error())
		}

		err = this.httpServer.ServeTLS(this.Listener, "", "")
		if err != nil && err != http.ErrServerClosed {
			return err
		}
	}

	return nil
}

func (this *HTTPListener) Close() error {
	if this.httpServer != nil {
		_ = this.httpServer.Close()
	}
	return this.Listener.Close()
}

func (this *HTTPListener) Reload(group *serverconfigs.ServerAddressGroup) {
	this.Group = group

	this.Reset()
}

// ServerHTTP 处理HTTP请求
func (this *HTTPListener) ServeHTTP(rawWriter http.ResponseWriter, rawReq *http.Request) {
	// 域名
	var reqHost = rawReq.Host

	// TLS域名
	if this.isIP(reqHost) {
		if rawReq.TLS != nil {
			serverName := rawReq.TLS.ServerName
			if len(serverName) > 0 {
				// 端口
				index := strings.LastIndex(reqHost, ":")
				if index >= 0 {
					reqHost = serverName + reqHost[index:]
				} else {
					reqHost = serverName
				}
			}
		}
	}

	// 防止空Host
	if len(reqHost) == 0 {
		ctx := rawReq.Context()
		if ctx != nil {
			addr := ctx.Value(http.LocalAddrContextKey)
			if addr != nil {
				reqHost = addr.(net.Addr).String()
			}
		}
	}

	domain, _, err := net.SplitHostPort(reqHost)
	if err != nil {
		domain = reqHost
	}

	server, serverName := this.findNamedServer(domain)
	if server == nil {
		if server == nil {
			this.handleMismatch(rawReq, rawWriter)
			return
		} else {
			serverName = domain
		}
	}

	// 包装新请求对象
	var req = &HTTPRequest{
		RawReq:     rawReq,
		RawWriter:  rawWriter,
		ReqServer:  server,
		ReqHost:    reqHost,
		ServerName: serverName,
		ServerAddr: this.addr,
		IsHTTP:     this.isHTTP,
		IsHTTPS:    this.isHTTPS,

		nodeConfig: sharedNodeConfig,
	}
	req.Do()
}

// 处理域名不匹配的情况
func (this *HTTPListener) handleMismatch(rawReq *http.Request, rawWriter http.ResponseWriter) {
	// TODO 需要记录访问记录和防止CC

	// 是否为健康检查
	var healthCheckKey = rawReq.Header.Get(serverconfigs.HealthCheckHeaderName)
	if len(healthCheckKey) > 0 {
		_, err := nodeutils.Base64DecodeMap(healthCheckKey)
		if err == nil {
			rawWriter.WriteHeader(http.StatusOK)
			return
		}
	}

	// 严格匹配域名模式下，我们拒绝用户访问
	if sharedNodeConfig.GlobalConfig != nil && sharedNodeConfig.GlobalConfig.HTTPAll.MatchDomainStrictly {
		var httpAllConfig = sharedNodeConfig.GlobalConfig.HTTPAll
		var mismatchAction = httpAllConfig.DomainMismatchAction
		if mismatchAction != nil && mismatchAction.Code == "page" {
			if mismatchAction.Options != nil {
				rawWriter.Header().Set("Content-Type", "text/html; charset=utf-8")
				rawWriter.WriteHeader(mismatchAction.Options.GetInt("statusCode"))
				_, _ = rawWriter.Write([]byte(mismatchAction.Options.GetString("contentHTML")))
			} else {
				http.Error(rawWriter, "404 page not found: '"+rawReq.URL.String()+"'", http.StatusNotFound)
			}
			return
		} else {
			hijacker, ok := rawWriter.(http.Hijacker)
			if ok {
				conn, _, _ := hijacker.Hijack()
				if conn != nil {
					_ = conn.Close()
					return
				}
			}
		}
	}

	http.Error(rawWriter, "404 page not found: '"+rawReq.URL.String()+"'", http.StatusNotFound)
}

func (this *HTTPListener) isIP(host string) bool {
	// IPv6
	if strings.Index(host, "[") > -1 {
		return true
	}

	for _, b := range host {
		if b >= 'a' && b <= 'z' {
			return false
		}
	}

	return true
}
