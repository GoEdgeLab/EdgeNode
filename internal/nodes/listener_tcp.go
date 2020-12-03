package nodes

import (
	"crypto/tls"
	"errors"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/logs"
	"net"
	"sync/atomic"
)

type TCPListener struct {
	BaseListener

	Listener net.Listener
}

func (this *TCPListener) Serve() error {
	listener := this.Listener
	if this.Group.IsTLS() {
		listener = tls.NewListener(listener, this.buildTLSConfig())
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			break
		}

		atomic.AddInt64(&this.countActiveConnections, 1)

		go func(conn net.Conn) {
			err = this.handleConn(conn)
			if err != nil {
				logs.Error("TCP_LISTENER", err.Error())
			}
			atomic.AddInt64(&this.countActiveConnections, -1)
		}(conn)
	}

	return nil
}

func (this *TCPListener) Reload(group *serverconfigs.ServerGroup) {
	this.Group = group
	this.Reset()
}

func (this *TCPListener) handleConn(conn net.Conn) error {
	firstServer := this.Group.FirstServer()
	if firstServer == nil {
		return errors.New("no server available")
	}
	if firstServer.ReverseProxy == nil {
		return errors.New("no ReverseProxy configured for the server")
	}
	originConn, err := this.connectOrigin(firstServer.ReverseProxy, conn.RemoteAddr().String())
	if err != nil {
		return err
	}

	var closer = func() {
		_ = conn.Close()
		_ = originConn.Close()
	}

	go func() {
		originBuffer := make([]byte, 32*1024) // TODO 需要可以设置，并可以使用Pool
		for {
			n, err := originConn.Read(originBuffer)
			if n > 0 {
				_, err = conn.Write(originBuffer[:n])
				if err != nil {
					closer()
					break
				}
			}
			if err != nil {
				closer()
				break
			}
		}
	}()

	clientBuffer := make([]byte, 32*1024) // TODO 需要可以设置，并可以使用Pool
	for {
		n, err := conn.Read(clientBuffer)
		if n > 0 {
			_, err = originConn.Write(clientBuffer[:n])
			if err != nil {
				break
			}
		}
		if err != nil {
			break
		}
	}

	// 关闭连接
	closer()

	return nil
}

func (this *TCPListener) Close() error {
	return this.Listener.Close()
}

func (this *TCPListener) connectOrigin(reverseProxy *serverconfigs.ReverseProxyConfig, remoteAddr string) (conn net.Conn, err error) {
	if reverseProxy == nil {
		return nil, errors.New("no reverse proxy config")
	}

	retries := 3
	for i := 0; i < retries; i++ {
		origin := reverseProxy.NextOrigin(nil)
		if origin == nil {
			continue
		}
		conn, err = OriginConnect(origin, remoteAddr)
		if err != nil {
			logs.Error("TCP_LISTENER", "unable to connect origin: "+origin.Addr.Host+":"+origin.Addr.PortRange+": "+err.Error())
			continue
		} else {
			return
		}
	}
	err = errors.New("no origin can be used")
	return
}
