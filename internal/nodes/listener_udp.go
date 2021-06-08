package nodes

import (
	"errors"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/stats"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"net"
	"sync"
	"time"
)

type UDPListener struct {
	BaseListener

	Listener *net.UDPConn

	connMap    map[string]*UDPConn
	connLocker sync.Mutex
	connTicker *utils.Ticker
}

func (this *UDPListener) Serve() error {
	firstServer := this.Group.FirstServer()
	if firstServer == nil {
		return errors.New("no server available")
	}
	if firstServer.ReverseProxy == nil {
		return errors.New("no ReverseProxy configured for the server")
	}

	this.connMap = map[string]*UDPConn{}
	this.connTicker = utils.NewTicker(1 * time.Minute)
	go func() {
		for this.connTicker.Next() {
			this.gcConns()
		}
	}()

	var buffer = make([]byte, 4*1024)
	for {
		n, addr, _ := this.Listener.ReadFrom(buffer)
		if n > 0 {
			this.connLocker.Lock()
			conn, ok := this.connMap[addr.String()]
			this.connLocker.Unlock()
			if ok && !conn.IsOk() {
				_ = conn.Close()
				ok = false
			}
			if !ok {
				originConn, err := this.connectOrigin(firstServer.ReverseProxy, "")
				if err != nil {
					remotelogs.Error("UDP_LISTENER", "unable to connect to origin server: "+err.Error())
					continue
				}
				if originConn == nil {
					remotelogs.Error("UDP_LISTENER", "unable to find a origin server")
					continue
				}
				conn = NewUDPConn(firstServer.Id, addr, this.Listener, originConn.(*net.UDPConn))
				this.connLocker.Lock()
				this.connMap[addr.String()] = conn
				this.connLocker.Unlock()
			}
			_, _ = conn.Write(buffer[:n])
		}
	}
}

func (this *UDPListener) Close() error {
	if this.connTicker != nil {
		this.connTicker.Stop()
	}

	// 关闭所有连接
	this.connLocker.Lock()
	for _, conn := range this.connMap {
		_ = conn.Close()
	}
	this.connLocker.Unlock()

	return this.Listener.Close()
}

func (this *UDPListener) Reload(group *serverconfigs.ServerGroup) {
	this.Group = group
	this.Reset()
}

func (this *UDPListener) connectOrigin(reverseProxy *serverconfigs.ReverseProxyConfig, remoteAddr string) (conn net.Conn, err error) {
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
			remotelogs.Error("UDP_LISTENER", "unable to connect origin: "+origin.Addr.Host+":"+origin.Addr.PortRange+": "+err.Error())
			continue
		} else {
			return
		}
	}
	err = errors.New("no origin can be used")
	return
}

// 回收连接
func (this *UDPListener) gcConns() {
	this.connLocker.Lock()
	closingConns := []*UDPConn{}
	for addr, conn := range this.connMap {
		if !conn.IsOk() {
			closingConns = append(closingConns, conn)
			delete(this.connMap, addr)
		}
	}
	this.connLocker.Unlock()

	for _, conn := range closingConns {
		_ = conn.Close()
	}
}

// UDPConn 自定义的UDP连接管理
type UDPConn struct {
	addr        net.Addr
	proxyConn   net.Conn
	serverConn  net.Conn
	activatedAt int64
	isOk        bool
	isClosed    bool
}

func NewUDPConn(serverId int64, addr net.Addr, proxyConn *net.UDPConn, serverConn *net.UDPConn) *UDPConn {
	conn := &UDPConn{
		addr:        addr,
		proxyConn:   proxyConn,
		serverConn:  serverConn,
		activatedAt: time.Now().Unix(),
		isOk:        true,
	}
	go func() {
		buffer := bytePool32k.Get()
		defer func() {
			bytePool32k.Put(buffer)
		}()

		for {
			n, err := serverConn.Read(buffer)
			if n > 0 {
				conn.activatedAt = time.Now().Unix()
				_, writingErr := proxyConn.WriteTo(buffer[:n], addr)
				if writingErr != nil {
					conn.isOk = false
					break
				}

				// 记录流量
				stats.SharedTrafficStatManager.Add(serverId, int64(n), 0, 0, 0)
			}
			if err != nil {
				conn.isOk = false
				break
			}
		}
	}()
	return conn
}

func (this *UDPConn) Write(b []byte) (n int, err error) {
	this.activatedAt = time.Now().Unix()
	n, err = this.serverConn.Write(b)
	if err != nil {
		this.isOk = false
	}
	return
}

func (this *UDPConn) Close() error {
	this.isOk = false
	if this.isClosed {
		return nil
	}
	this.isClosed = true
	return this.serverConn.Close()
}

func (this *UDPConn) IsOk() bool {
	if !this.isOk {
		return false
	}
	return time.Now().Unix()-this.activatedAt < 30 // 如果超过 N 秒没有活动我们认为是超时
}
