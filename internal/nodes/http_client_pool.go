package nodes

import (
	"context"
	"crypto/tls"
	"errors"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"net"
	"net/http"
	"runtime"
	"strconv"
	"sync"
	"time"
)

// SharedHTTPClientPool HTTP客户端池单例
var SharedHTTPClientPool = NewHTTPClientPool()

// HTTPClientPool 客户端池
type HTTPClientPool struct {
	clientExpiredDuration time.Duration
	clientsMap            map[string]*HTTPClient // backend key => client
	locker                sync.Mutex
}

// NewHTTPClientPool 获取新对象
func NewHTTPClientPool() *HTTPClientPool {
	pool := &HTTPClientPool{
		clientExpiredDuration: 3600 * time.Second,
		clientsMap:            map[string]*HTTPClient{},
	}

	go pool.cleanClients()

	return pool
}

// Client 根据地址获取客户端
func (this *HTTPClientPool) Client(req *http.Request, origin *serverconfigs.OriginConfig, originAddr string) (rawClient *http.Client, err error) {
	if origin.Addr == nil {
		return nil, errors.New("origin addr should not be empty (originId:" + strconv.FormatInt(origin.Id, 10) + ")")
	}

	key := origin.UniqueKey() + "@" + originAddr

	this.locker.Lock()
	defer this.locker.Unlock()

	client, found := this.clientsMap[key]
	if found {
		client.UpdateAccessTime()
		return client.RawClient(), nil
	}

	maxConnections := origin.MaxConns
	connectionTimeout := origin.ConnTimeoutDuration()
	readTimeout := origin.ReadTimeoutDuration()
	idleTimeout := origin.IdleTimeoutDuration()
	idleConns := origin.MaxIdleConns

	// 超时时间
	if connectionTimeout <= 0 {
		connectionTimeout = 15 * time.Second
	}

	if idleTimeout <= 0 {
		idleTimeout = 2 * time.Minute
	}

	numberCPU := runtime.NumCPU()
	if numberCPU < 8 {
		numberCPU = 8
	}
	if maxConnections <= 0 {
		maxConnections = numberCPU * 8
	}

	if idleConns <= 0 {
		idleConns = numberCPU * 4
	}
	//logs.Println("[ORIGIN]max connections:", maxConnections)

	// TLS通讯
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}
	if origin.Cert != nil {
		obj := origin.Cert.CertObject()
		if obj != nil {
			tlsConfig.InsecureSkipVerify = false
			tlsConfig.Certificates = []tls.Certificate{*obj}
			if len(origin.Cert.ServerName) > 0 {
				tlsConfig.ServerName = origin.Cert.ServerName
			}
		}
	}

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			// 支持TOA的连接
			toaConfig := sharedTOAManager.Config()
			if toaConfig != nil && toaConfig.IsOn {
				retries := 3
				for i := 1; i <= retries; i++ {
					port := int(toaConfig.RandLocalPort())
					// TODO 思考是否支持X-Real-IP/X-Forwarded-IP
					err := sharedTOAManager.SendMsg("add:" + strconv.Itoa(port) + ":" + req.RemoteAddr)
					if err != nil {
						remotelogs.Error("TOA", "add failed: "+err.Error())
					} else {
						dialer := net.Dialer{
							Timeout:   connectionTimeout,
							KeepAlive: 1 * time.Minute,
							LocalAddr: &net.TCPAddr{
								Port: port,
							},
						}
						conn, err := dialer.DialContext(ctx, network, originAddr)
						// TODO 需要在合适的时机删除TOA记录
						if err == nil || i == retries {
							return conn, err
						}
					}
				}
			}

			// 普通的连接
			return (&net.Dialer{
				Timeout:   connectionTimeout,
				KeepAlive: 1 * time.Minute,
			}).DialContext(ctx, network, originAddr)
		},
		MaxIdleConns:          0,
		MaxIdleConnsPerHost:   idleConns,
		MaxConnsPerHost:       maxConnections,
		IdleConnTimeout:       idleTimeout,
		ExpectContinueTimeout: 1 * time.Second,
		TLSHandshakeTimeout:   0, // 不限
		TLSClientConfig:       tlsConfig,
		Proxy:                 nil,
	}

	rawClient = &http.Client{
		Timeout:   readTimeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	this.clientsMap[key] = NewHTTPClient(rawClient)

	return rawClient, nil
}

// 清理不使用的Client
func (this *HTTPClientPool) cleanClients() {
	ticker := time.NewTicker(this.clientExpiredDuration)
	for range ticker.C {
		currentAt := time.Now().Unix()

		this.locker.Lock()
		for k, client := range this.clientsMap {
			if client.AccessTime() < currentAt+86400 { // 超过 N 秒没有调用就关闭
				delete(this.clientsMap, k)
				client.Close()
			}
		}
		this.locker.Unlock()
	}
}
