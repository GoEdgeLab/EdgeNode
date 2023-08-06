package nodes

import (
	"context"
	"crypto/tls"
	"errors"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/TeaOSLab/EdgeNode/internal/utils/fasttime"
	"github.com/pires/go-proxyproto"
	"golang.org/x/net/http2"
	"net"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// SharedHTTPClientPool HTTP客户端池单例
var SharedHTTPClientPool = NewHTTPClientPool()

const httpClientProxyProtocolTag = "@ProxyProtocol@"

// HTTPClientPool 客户端池
type HTTPClientPool struct {
	clientsMap map[string]*HTTPClient // backend key => client

	cleanTicker *time.Ticker

	locker sync.RWMutex
}

// NewHTTPClientPool 获取新对象
func NewHTTPClientPool() *HTTPClientPool {
	var pool = &HTTPClientPool{
		cleanTicker: time.NewTicker(1 * time.Hour),
		clientsMap:  map[string]*HTTPClient{},
	}

	goman.New(func() {
		pool.cleanClients()
	})

	return pool
}

// Client 根据地址获取客户端
func (this *HTTPClientPool) Client(req *HTTPRequest,
	origin *serverconfigs.OriginConfig,
	originAddr string,
	proxyProtocol *serverconfigs.ProxyProtocolConfig,
	followRedirects bool) (rawClient *http.Client, err error) {
	if origin.Addr == nil {
		return nil, errors.New("origin addr should not be empty (originId:" + strconv.FormatInt(origin.Id, 10) + ")")
	}

	var key = origin.UniqueKey() + "@" + originAddr

	// if we are under available ProxyProtocol, we add client ip to key to make every client unique
	var isProxyProtocol = false
	if proxyProtocol != nil && proxyProtocol.IsOn {
		key += httpClientProxyProtocolTag + req.requestRemoteAddr(true)
		isProxyProtocol = true
	}

	var isLnRequest = origin.Id == 0

	this.locker.RLock()
	client, found := this.clientsMap[key]
	this.locker.RUnlock()
	if found {
		client.UpdateAccessTime()
		return client.RawClient(), nil
	}

	// 这里不能使用RLock，避免因为并发生成多个同样的client实例
	this.locker.Lock()
	defer this.locker.Unlock()

	// 再次查找
	client, found = this.clientsMap[key]
	if found {
		client.UpdateAccessTime()
		return client.RawClient(), nil
	}

	var maxConnections = origin.MaxConns
	var connectionTimeout = origin.ConnTimeoutDuration()
	var readTimeout = origin.ReadTimeoutDuration()
	var idleTimeout = origin.IdleTimeoutDuration()
	var idleConns = origin.MaxIdleConns

	// 超时时间
	if connectionTimeout <= 0 {
		connectionTimeout = 15 * time.Second
	}

	if idleTimeout <= 0 {
		idleTimeout = 2 * time.Minute
	}

	var numberCPU = runtime.NumCPU()
	if numberCPU < 8 {
		numberCPU = 8
	}
	if maxConnections <= 0 {
		maxConnections = numberCPU * 64
	}

	if idleConns <= 0 {
		idleConns = numberCPU * 16
	}

	if isProxyProtocol { // ProxyProtocol无需保持太多空闲连接
		idleConns = 3
	} else if isLnRequest { // 可以判断为Ln节点请求
		maxConnections *= 8
		idleConns *= 8
		idleTimeout *= 4
	} else if sharedNodeConfig != nil && sharedNodeConfig.Level > 1 {
		// Ln节点可以适当增加连接数
		maxConnections *= 2
		idleConns *= 2
	}

	// TLS通讯
	var tlsConfig = &tls.Config{
		InsecureSkipVerify: true,
	}
	if origin.Cert != nil {
		var obj = origin.Cert.CertObject()
		if obj != nil {
			tlsConfig.InsecureSkipVerify = false
			tlsConfig.Certificates = []tls.Certificate{*obj}
			if len(origin.Cert.ServerName) > 0 {
				tlsConfig.ServerName = origin.Cert.ServerName
			}
		}
	}

	var transport = &HTTPClientTransport{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				// 普通的连接
				conn, err := (&net.Dialer{
					Timeout:   connectionTimeout,
					KeepAlive: 1 * time.Minute,
				}).DialContext(ctx, network, originAddr)
				if err != nil {
					return nil, err
				}

				// 处理PROXY protocol
				err = this.handlePROXYProtocol(conn, req, proxyProtocol)
				if err != nil {
					return nil, err
				}

				return NewOriginConn(conn), nil
			},
			MaxIdleConns:          0,
			MaxIdleConnsPerHost:   idleConns,
			MaxConnsPerHost:       maxConnections,
			IdleConnTimeout:       idleTimeout,
			ExpectContinueTimeout: 1 * time.Second,
			TLSHandshakeTimeout:   5 * time.Second,
			TLSClientConfig:       tlsConfig,
			ReadBufferSize:        8 * 1024,
			Proxy:                 nil,
		},
	}

	// support http/2
	if origin.HTTP2Enabled && origin.Addr != nil && origin.Addr.Protocol == serverconfigs.ProtocolHTTPS {
		_ = http2.ConfigureTransport(transport.Transport)
	}

	rawClient = &http.Client{
		Timeout:   readTimeout,
		Transport: transport,
		CheckRedirect: func(targetReq *http.Request, via []*http.Request) error {
			// 是否跟随
			if followRedirects {
				var schemeIsSame = true
				for _, r := range via {
					if r.URL.Scheme != targetReq.URL.Scheme {
						schemeIsSame = false
						break
					}
				}
				if schemeIsSame {
					return nil
				}
			}

			return http.ErrUseLastResponse
		},
	}

	this.clientsMap[key] = NewHTTPClient(rawClient)

	return rawClient, nil
}

// 清理不使用的Client
func (this *HTTPClientPool) cleanClients() {
	for range this.cleanTicker.C {
		var nowTime = fasttime.Now().Unix()

		var expiredKeys = []string{}
		var expiredClients = []*HTTPClient{}

		// lookup expired clients
		this.locker.RLock()
		for k, client := range this.clientsMap {
			if client.AccessTime() < nowTime-86400 ||
				(strings.Contains(k, httpClientProxyProtocolTag) && client.AccessTime() < nowTime-3600) { // 超过 N 秒没有调用就关闭
				expiredKeys = append(expiredKeys, k)
				expiredClients = append(expiredClients, client)
			}
		}
		this.locker.RUnlock()

		// remove expired keys
		if len(expiredKeys) > 0 {
			this.locker.Lock()
			for _, k := range expiredKeys {
				delete(this.clientsMap, k)
			}
			this.locker.Unlock()
		}

		// close expired clients
		if len(expiredClients) > 0 {
			for _, client := range expiredClients {
				client.Close()
			}
		}
	}
}

// 支持PROXY Protocol
func (this *HTTPClientPool) handlePROXYProtocol(conn net.Conn, req *HTTPRequest, proxyProtocol *serverconfigs.ProxyProtocolConfig) error {
	if proxyProtocol != nil &&
		proxyProtocol.IsOn &&
		(proxyProtocol.Version == serverconfigs.ProxyProtocolVersion1 || proxyProtocol.Version == serverconfigs.ProxyProtocolVersion2) {
		var remoteAddr = req.requestRemoteAddr(true)
		var transportProtocol = proxyproto.TCPv4
		if strings.Contains(remoteAddr, ":") {
			transportProtocol = proxyproto.TCPv6
		}
		var destAddr = conn.RemoteAddr()
		var reqConn = req.RawReq.Context().Value(HTTPConnContextKey)
		if reqConn != nil {
			destAddr = reqConn.(net.Conn).LocalAddr()
		}
		var header = proxyproto.Header{
			Version:           byte(proxyProtocol.Version),
			Command:           proxyproto.PROXY,
			TransportProtocol: transportProtocol,
			SourceAddr: &net.TCPAddr{
				IP:   net.ParseIP(remoteAddr),
				Port: req.requestRemotePort(),
			},
			DestinationAddr: destAddr,
		}
		_, err := header.WriteTo(conn)
		if err != nil {
			_ = conn.Close()
			return err
		}
		return nil
	}

	return nil
}
