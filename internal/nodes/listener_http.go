package nodes

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/iwind/TeaGo/logs"
	"golang.org/x/net/http2"
	"net"
	"net/http"
	"time"
)

type HTTPListener struct {
	BaseListener

	Group    *serverconfigs.ServerGroup
	Listener net.Listener

	httpServer *http.Server
}

func (this *HTTPListener) Serve() error {
	handler := http.NewServeMux()
	handler.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		this.handleHTTP(writer, request)
	})

	this.httpServer = &http.Server{
		Addr:        this.Group.Addr(),
		Handler:     handler,
		IdleTimeout: 2 * time.Minute,
	}
	this.httpServer.SetKeepAlivesEnabled(true)

	// HTTP协议
	if this.Group.IsHTTP() {
		err := this.httpServer.Serve(this.Listener)
		if err != nil && err != http.ErrServerClosed {
			return err
		}
	}

	// HTTPS协议
	if this.Group.IsHTTPS() {
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

func (this *HTTPListener) handleHTTP(writer http.ResponseWriter, req *http.Request) {
	writer.Write([]byte("Hello, World"))
}
