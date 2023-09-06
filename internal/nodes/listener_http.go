package nodes

import (
	"context"
	"crypto/tls"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/iwind/TeaGo/Tea"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

var httpErrorLogger = log.New(io.Discard, "", 0)

const HTTPIdleTimeout = 75 * time.Second

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
	isHTTP3    bool
	httpServer *http.Server
}

func (this *HTTPListener) Serve() error {
	this.addr = this.Group.Addr()
	this.isHTTP = this.Group.IsHTTP()
	this.isHTTPS = this.Group.IsHTTPS()

	this.httpServer = &http.Server{
		Addr:              this.addr,
		Handler:           this,
		ReadHeaderTimeout: 3 * time.Second, // TODO 改成可以配置
		IdleTimeout:       HTTPIdleTimeout, // TODO 改成可以配置
		ConnState: func(conn net.Conn, state http.ConnState) {
			switch state {
			case http.StateNew:
				atomic.AddInt64(&this.countActiveConnections, 1)
			case http.StateClosed:
				atomic.AddInt64(&this.countActiveConnections, -1)
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

		err := this.httpServer.ServeTLS(this.Listener, "", "")
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
	var globalServerConfig = sharedNodeConfig.GlobalServerConfig
	if globalServerConfig != nil && !globalServerConfig.HTTPAll.SupportsLowVersionHTTP && (rawReq.ProtoMajor < 1 /** 0.x **/ || (rawReq.ProtoMajor == 1 && rawReq.ProtoMinor == 0 /** 1.0 **/)) {
		http.Error(rawWriter, rawReq.Proto+" request is not supported.", http.StatusBadRequest)
		return
	}

	// 不支持Connect
	if rawReq.Method == http.MethodConnect {
		http.Error(rawWriter, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// 域名
	var reqHost = strings.ToLower(strings.TrimRight(rawReq.Host, "."))

	// TLS域名
	if this.isIP(reqHost) {
		if rawReq.TLS != nil {
			var serverName = rawReq.TLS.ServerName
			if len(serverName) > 0 {
				// 端口
				var index = strings.LastIndex(reqHost, ":")
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
		var ctx = rawReq.Context()
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
			// 增加默认的一个服务
			server = this.emptyServer()
		} else {
			serverName = domain
		}
	} else if !server.CNameAsDomain && server.CNameDomain == domain {
		server = this.emptyServer()
	}

	// 绑定连接
	if server != nil && server.Id > 0 {
		var requestConn = rawReq.Context().Value(HTTPConnContextKey)
		if requestConn != nil {
			clientConn, ok := requestConn.(ClientConnInterface)
			if ok {
				var goNext = clientConn.SetServerId(server.Id)
				if !goNext {
					return
				}
				clientConn.SetUserId(server.UserId)

				var userPlanId int64
				if server.UserPlan != nil && server.UserPlan.Id > 0 {
					userPlanId = server.UserPlan.Id
				}
				clientConn.SetUserPlanId(userPlanId)
			}
		}
	}

	// 检查用户
	if server != nil && server.UserId > 0 {
		if !SharedUserManager.CheckUserServersIsEnabled(server.UserId) {
			rawWriter.WriteHeader(http.StatusNotFound)
			_, _ = rawWriter.Write([]byte("The site owner is unavailable."))
			return
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
		IsHTTP3:    this.isHTTP3,

		nodeConfig: sharedNodeConfig,
	}
	req.Do()
}

// 检查host是否为IP
func (this *HTTPListener) isIP(host string) bool {
	// IPv6
	if strings.Contains(host, "[") {
		return true
	}

	for _, b := range host {
		if b >= 'a' && b <= 'z' {
			return false
		}
	}

	return true
}

// 默认的访问日志
func (this *HTTPListener) emptyServer() *serverconfigs.ServerConfig {
	var server = &serverconfigs.ServerConfig{
		Type: serverconfigs.ServerTypeHTTPProxy,
	}

	// 检查是否开启访问日志
	if sharedNodeConfig != nil {
		var globalServerConfig = sharedNodeConfig.GlobalServerConfig
		if globalServerConfig != nil && globalServerConfig.HTTPAccessLog.EnableServerNotFound {
			var accessLogRef = serverconfigs.NewHTTPAccessLogRef()
			accessLogRef.IsOn = true
			accessLogRef.Fields = append([]int{}, serverconfigs.HTTPAccessLogDefaultFieldsCodes...)
			server.Web = &serverconfigs.HTTPWebConfig{
				IsOn:         true,
				AccessLogRef: accessLogRef,
			}
		}
	}

	// TODO 需要对访问频率过多的IP进行惩罚

	return server
}
