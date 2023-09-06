package nodes

import (
	"errors"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/stats"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/iwind/TeaGo/types"
	"github.com/pires/go-proxyproto"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
	"net"
	"strings"
	"sync"
	"time"
)

const (
	UDPConnLifeSeconds = 30
)

type UDPPacketListener interface {
	ReadFrom(b []byte) (n int, cm any, src net.Addr, err error)
	WriteTo(b []byte, cm any, dst net.Addr) (n int, err error)
	LocalAddr() net.Addr
}

type UDPIPv4Listener struct {
	rawListener *ipv4.PacketConn
}

func NewUDPIPv4Listener(rawListener *ipv4.PacketConn) *UDPIPv4Listener {
	return &UDPIPv4Listener{rawListener: rawListener}
}

func (this *UDPIPv4Listener) ReadFrom(b []byte) (n int, cm any, src net.Addr, err error) {
	return this.rawListener.ReadFrom(b)
}

func (this *UDPIPv4Listener) WriteTo(b []byte, cm any, dst net.Addr) (n int, err error) {
	return this.rawListener.WriteTo(b, cm.(*ipv4.ControlMessage), dst)
}

func (this *UDPIPv4Listener) LocalAddr() net.Addr {
	return this.rawListener.LocalAddr()
}

type UDPIPv6Listener struct {
	rawListener *ipv6.PacketConn
}

func NewUDPIPv6Listener(rawListener *ipv6.PacketConn) *UDPIPv6Listener {
	return &UDPIPv6Listener{rawListener: rawListener}
}

func (this *UDPIPv6Listener) ReadFrom(b []byte) (n int, cm any, src net.Addr, err error) {
	return this.rawListener.ReadFrom(b)
}

func (this *UDPIPv6Listener) WriteTo(b []byte, cm any, dst net.Addr) (n int, err error) {
	return this.rawListener.WriteTo(b, cm.(*ipv6.ControlMessage), dst)
}

func (this *UDPIPv6Listener) LocalAddr() net.Addr {
	return this.rawListener.LocalAddr()
}

type UDPListener struct {
	BaseListener

	IPv4Listener *ipv4.PacketConn
	IPv6Listener *ipv6.PacketConn

	connMap    map[string]*UDPConn
	connLocker sync.Mutex
	connTicker *utils.Ticker

	reverseProxy *serverconfigs.ReverseProxyConfig

	port int

	isClosed bool
}

func (this *UDPListener) Serve() error {
	if this.Group == nil {
		return nil
	}
	var server = this.Group.FirstServer()
	if server == nil {
		return nil
	}
	var serverId = server.Id

	var wg = &sync.WaitGroup{}
	wg.Add(2) // 2 = ipv4 + ipv6

	go func() {
		defer wg.Done()

		if this.IPv4Listener != nil {
			err := this.IPv4Listener.SetControlMessage(ipv4.FlagDst, true)
			if err != nil {
				remotelogs.ServerError(serverId, "UDP_LISTENER", "can not serve ipv4 listener: "+err.Error(), "", nil)
				return
			}

			err = this.servePacketListener(NewUDPIPv4Listener(this.IPv4Listener))
			if err != nil {
				remotelogs.ServerError(serverId, "UDP_LISTENER", "can not serve ipv4 listener: "+err.Error(), "", nil)
				return
			}
		}
	}()

	go func() {
		defer wg.Done()

		if this.IPv6Listener != nil {
			err := this.IPv6Listener.SetControlMessage(ipv6.FlagDst, true)
			if err != nil {
				remotelogs.ServerError(serverId, "UDP_LISTENER", "can not serve ipv6 listener: "+err.Error(), "", nil)
				return
			}

			err = this.servePacketListener(NewUDPIPv6Listener(this.IPv6Listener))
			if err != nil {
				remotelogs.ServerError(serverId, "UDP_LISTENER", "can not serve ipv6 listener: "+err.Error(), "", nil)
				return
			}
		}
	}()

	wg.Wait()

	return nil
}

func (this *UDPListener) servePacketListener(listener UDPPacketListener) error {
	// 获取分组端口
	var groupAddr = this.Group.Addr()
	var portIndex = strings.LastIndex(groupAddr, ":")
	if portIndex >= 0 {
		var port = groupAddr[portIndex+1:]
		this.port = types.Int(port)
	}

	var firstServer = this.Group.FirstServer()
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

		// 检查用户状态
		if firstServer.UserId > 0 && !SharedUserManager.CheckUserServersIsEnabled(firstServer.UserId) {
			return nil
		}

		n, cm, clientAddr, err := listener.ReadFrom(buffer)
		if err != nil {
			if this.isClosed {
				return nil
			}
			return err
		}

		if n > 0 {
			this.connLocker.Lock()
			conn, ok := this.connMap[clientAddr.String()]
			this.connLocker.Unlock()
			if ok && !conn.IsOk() {
				_ = conn.Close()
				ok = false
			}
			if !ok {
				originConn, err := this.connectOrigin(firstServer.Id, this.reverseProxy, listener.LocalAddr(), clientAddr)
				if err != nil {
					remotelogs.Error("UDP_LISTENER", "unable to connect to origin server: "+err.Error())
					continue
				}
				if originConn == nil {
					remotelogs.Error("UDP_LISTENER", "unable to find a origin server")
					continue
				}
				conn = NewUDPConn(firstServer, clientAddr, listener, cm, originConn.(*net.UDPConn))
				this.connLocker.Lock()
				this.connMap[clientAddr.String()] = conn
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

	var errorStrings = []string{}
	if this.IPv4Listener != nil {
		err := this.IPv4Listener.Close()
		if err != nil {
			errorStrings = append(errorStrings, err.Error())
		}
	}

	if this.IPv6Listener != nil {
		err := this.IPv6Listener.Close()
		if err != nil {
			errorStrings = append(errorStrings, err.Error())
		}
	}

	if len(errorStrings) > 0 {
		return errors.New(errorStrings[0])
	}

	return nil
}

func (this *UDPListener) Reload(group *serverconfigs.ServerAddressGroup) {
	this.Group = group
	this.Reset()

	// 重置配置
	var firstServer = this.Group.FirstServer()
	if firstServer == nil {
		return
	}
	this.reverseProxy = firstServer.ReverseProxy
}

func (this *UDPListener) connectOrigin(serverId int64, reverseProxy *serverconfigs.ReverseProxyConfig, localAddr net.Addr, remoteAddr net.Addr) (conn net.Conn, err error) {
	if reverseProxy == nil {
		return nil, errors.New("no reverse proxy config")
	}

	var retries = 3
	var addr string

	var failedOriginIds []int64

	for i := 0; i < retries; i++ {
		var origin *serverconfigs.OriginConfig
		if len(failedOriginIds) > 0 {
			origin = reverseProxy.AnyOrigin(nil, failedOriginIds)
		}
		if origin == nil {
			origin = reverseProxy.NextOrigin(nil)
		}
		if origin == nil {
			continue
		}

		conn, addr, err = OriginConnect(origin, this.port, remoteAddr.String(), "")
		if err != nil {
			failedOriginIds = append(failedOriginIds, origin.Id)

			remotelogs.ServerError(serverId, "UDP_LISTENER", "unable to connect origin server: "+addr+": "+err.Error(), "", nil)

			SharedOriginStateManager.Fail(origin, "", reverseProxy, func() {
				reverseProxy.ResetScheduling()
			})

			continue
		} else {
			if !origin.IsOk {
				SharedOriginStateManager.Success(origin, func() {
					reverseProxy.ResetScheduling()
				})
			}

			// PROXY Protocol
			if reverseProxy != nil &&
				reverseProxy.ProxyProtocol != nil &&
				reverseProxy.ProxyProtocol.IsOn &&
				(reverseProxy.ProxyProtocol.Version == serverconfigs.ProxyProtocolVersion1 || reverseProxy.ProxyProtocol.Version == serverconfigs.ProxyProtocolVersion2) {
				var transportProtocol = proxyproto.UDPv4
				if strings.Contains(remoteAddr.String(), "[") {
					transportProtocol = proxyproto.UDPv6
				}
				var header = proxyproto.Header{
					Version:           byte(reverseProxy.ProxyProtocol.Version),
					Command:           proxyproto.PROXY,
					TransportProtocol: transportProtocol,
					SourceAddr:        remoteAddr,
					DestinationAddr:   localAddr,
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

	if err == nil {
		err = errors.New("server '" + types.String(serverId) + "': no available origin server can be used")
	}
	return
}

// 回收连接
func (this *UDPListener) gcConns() {
	this.connLocker.Lock()
	var closingConns = []*UDPConn{}
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
	addr          net.Addr
	proxyListener UDPPacketListener
	serverConn    net.Conn
	activatedAt   int64
	isOk          bool
	isClosed      bool
}

func NewUDPConn(server *serverconfigs.ServerConfig, addr net.Addr, proxyListener UDPPacketListener, cm any, serverConn *net.UDPConn) *UDPConn {
	var conn = &UDPConn{
		addr:          addr,
		proxyListener: proxyListener,
		serverConn:    serverConn,
		activatedAt:   time.Now().Unix(),
		isOk:          true,
	}

	// 统计
	if server != nil {
		stats.SharedTrafficStatManager.Add(server.UserId, server.Id, "", 0, 0, 1, 0, 0, 0, server.ShouldCheckTrafficLimit(), server.PlanId())
	}

	// 处理ControlMessage
	switch controlMessage := cm.(type) {
	case *ipv4.ControlMessage:
		controlMessage.Src = controlMessage.Dst
	case *ipv6.ControlMessage:
		controlMessage.Src = controlMessage.Dst
	}

	goman.New(func() {
		var buffer = utils.BytePool4k.Get()
		defer func() {
			utils.BytePool4k.Put(buffer)
		}()

		for {
			n, err := serverConn.Read(buffer)
			if n > 0 {
				conn.activatedAt = time.Now().Unix()

				_, writingErr := proxyListener.WriteTo(buffer[:n], cm, addr)
				if writingErr != nil {
					conn.isOk = false
					break
				}

				// 记录流量和带宽
				if server != nil {
					// 流量
					stats.SharedTrafficStatManager.Add(server.UserId, server.Id, "", int64(n), 0, 0, 0, 0, 0, server.ShouldCheckTrafficLimit(), server.PlanId())

					// 带宽
					var userPlanId int64
					if server.UserPlan != nil && server.UserPlan.Id > 0 {
						userPlanId = server.UserPlan.Id
					}
					stats.SharedBandwidthStatManager.AddBandwidth(server.UserId, userPlanId, server.Id, int64(n), int64(n))
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
	return time.Now().Unix()-this.activatedAt < UDPConnLifeSeconds // 如果超过 N 秒没有活动我们认为是超时
}
