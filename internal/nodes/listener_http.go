package nodes

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/iwind/TeaGo/logs"
	"golang.org/x/net/http2"
	"net"
	"net/http"
	"strings"
	"time"
)

type HTTPListener struct {
	BaseListener

	Group    *serverconfigs.ServerGroup
	Listener net.Listener

	addr       string
	isHTTP     bool
	isHTTPS    bool
	httpServer *http.Server
}

func (this *HTTPListener) Serve() error {
	handler := http.NewServeMux()
	handler.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		this.handleHTTP(writer, request)
	})

	this.addr = this.Group.Addr()
	this.isHTTP = this.Group.IsHTTP()
	this.isHTTPS = this.Group.IsHTTPS()

	this.httpServer = &http.Server{
		Addr:        this.addr,
		Handler:     handler,
		IdleTimeout: 2 * time.Minute,
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
		this.httpServer.TLSConfig = this.buildTLSConfig(this.Group)

		// support http/2
		err := http2.ConfigureServer(this.httpServer, nil)
		if err != nil {
			logs.Println("[HTTP_LISTENER]configure http2 error: " + err.Error())
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

func (this *HTTPListener) Reload(group *serverconfigs.ServerGroup) {
	this.Group = group

	if this.isHTTPS {
		this.httpServer.TLSConfig = this.buildTLSConfig(this.Group)
	}

	this.Reset()
}

// 处理HTTP请求
func (this *HTTPListener) handleHTTP(rawWriter http.ResponseWriter, rawReq *http.Request) {
	// 域名
	reqHost := rawReq.Host

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

	server, serverName := this.findNamedServer(this.Group, domain)
	if server == nil {
		// 严格匹配域名模式下，我们拒绝用户访问
		if sharedNodeConfig.GlobalConfig != nil && sharedNodeConfig.GlobalConfig.HTTPAll.MatchDomainStrictly {
			hijacker, ok := rawWriter.(http.Hijacker)
			if ok {
				conn, _, _ := hijacker.Hijack()
				if conn != nil {
					_ = conn.Close()
					return
				}
			}
		}

		http.Error(rawWriter, "404 page not found: '"+rawReq.URL.String()+"'", http.StatusNotFound)
		return
	}

	// 包装新请求对象
	req := &HTTPRequest{
		RawReq:     rawReq,
		RawWriter:  rawWriter,
		Server:     server,
		Host:       reqHost,
		ServerName: serverName,
		ServerAddr: this.addr,
		IsHTTP:     this.isHTTP,
		IsHTTPS:    this.isHTTPS,
	}
	req.Do()
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
