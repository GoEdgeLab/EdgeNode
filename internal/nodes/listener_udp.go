package nodes

import (
	"errors"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/stats"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/pires/go-proxyproto"
	"net"
	"strings"
	"sync"
	"time"
)

type UDPListener struct {
	BaseListener

	Listener *net.UDPConn

	connMap    map[string]*UDPConn
	connLocker sync.Mutex
	connTicker *utils.Ticker

	reverseProxy *serverconfigs.ReverseProxyConfig

	isClosed bool
}

func (this *UDPListener) Serve() error {
	firstServer := this.Group.FirstServer()
	if firstServer == nil {
		return errors.New("no server available")
	}
	this.reverseProxy = firstServer.ReverseProxy
	if this.reverseProxy == nil {
		return errors.New("no ReverseProxy configured for the server '" + firstServer.Name + "'")
	}

	this.connMap = map[string]*UDPConn{}
	this.connTicker = utils.NewTicker(1 * time.Minute)
	goman.New(func() {
		for this.connTicker.Next() {
			this.gcConns()
		}
	})

	var buffer = make([]byte, 4*1024)
	for {
		if this.isClosed {
			return nil
		}

		n, addr, err := this.Listener.ReadFrom(buffer)
		if err != nil {
			if this.isClosed {
				return nil
			}
			return err
		}

		if n > 0 {
			this.connLocker.Lock()
			conn, ok := this.connMap[addr.String()]
			this.connLocker.Unlock()
			if ok && !conn.IsOk() {
				_ = conn.Close()
				ok = false
			}
			if !ok {
				originConn, err := this.connectOrigin(firstServer.Id, this.reverseProxy, addr)
				if err != nil {
					remotelogs.Error("UDP_LISTENER", "unable to connect to origin server: "+err.Error())
					continue
				}
				if originConn == nil {
					remotelogs.Error("UDP_LISTENER", "unable to find a origin server")
					continue
				}
				conn = NewUDPConn(firstServer, addr, this.Listener, originConn.(*net.UDPConn))
				this.connLocker.Lock()
				this.connMap[addr.String()] = conn
				this.connLocker.Unlock()
			}
			_, _ = conn.Write(buffer[:n])
		}
	}
}

func (this *UDPListener) Close() error {
	this.isClosed = true

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

func (this *UDPListener) Reload(group *serverconfigs.ServerAddressGroup) {
	this.Group = group
	this.Reset()

	// 重置配置
	firstServer := this.Group.FirstServer()
	if firstServer == nil {
		return
	}
	this.reverseProxy = firstServer.ReverseProxy
}

func (this *UDPListener) connectOrigin(serverId int64, reverseProxy *serverconfigs.ReverseProxyConfig, remoteAddr net.Addr) (conn net.Conn, err error) {
	if reverseProxy == nil {
		return nil, errors.New("no reverse proxy config")
	}

	retries := 3
	for i := 0; i < retries; i++ {
		origin := reverseProxy.NextOrigin(nil)
		if origin == nil {
			continue
		}
		conn, err = OriginConnect(origin, remoteAddr.String())
		if err != nil {
			remotelogs.ServerError(serverId, "UDP_LISTENER", "unable to connect origin: "+origin.Addr.Host+":"+origin.Addr.PortRange+": "+err.Error(), "", nil)
			continue
		} else {
			// PROXY Protocol
			if reverseProxy != nil &&
				reverseProxy.ProxyProtocol != nil &&
				reverseProxy.ProxyProtocol.IsOn &&
				(reverseProxy.ProxyProtocol.Version == serverconfigs.ProxyProtocolVersion1 || reverseProxy.ProxyProtocol.Version == serverconfigs.ProxyProtocolVersion2) {
				var transportProtocol = proxyproto.UDPv4
				if strings.Contains(remoteAddr.String(), "[") {
					transportProtocol = proxyproto.UDPv6
				}
				header := proxyproto.Header{
					Version:           byte(reverseProxy.ProxyProtocol.Version),
					Command:           proxyproto.PROXY,
					TransportProtocol: transportProtocol,
					SourceAddr:        remoteAddr,
					DestinationAddr:   this.Listener.LocalAddr(),
				}
				_, err = header.WriteTo(conn)
				if err != nil {
					_ = conn.Close()
					return nil, err
				}
			}

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

func NewUDPConn(server *serverconfigs.ServerConfig, addr net.Addr, proxyConn *net.UDPConn, serverConn *net.UDPConn) *UDPConn {
	conn := &UDPConn{
		addr:        addr,
		proxyConn:   proxyConn,
		serverConn:  serverConn,
		activatedAt: time.Now().Unix(),
		isOk:        true,
	}

	// 统计
	if server != nil {
		stats.SharedTrafficStatManager.Add(server.Id, "", 0, 0, 1, 0, 0, 0, server.ShouldCheckTrafficLimit(), server.PlanId())
	}

	goman.New(func() {
		buffer := utils.BytePool4k.Get()
		defer func() {
			utils.BytePool4k.Put(buffer)
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
				if server != nil {
					stats.SharedTrafficStatManager.Add(server.Id, "", int64(n), 0, 0, 0, 0, 0, server.ShouldCheckTrafficLimit(), server.PlanId())
				}
			}
			if err != nil {
				conn.isOk = false
				break
			}
		}
	})
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
