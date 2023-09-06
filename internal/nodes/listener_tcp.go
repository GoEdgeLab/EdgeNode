package nodes

import (
	"crypto/tls"
	"errors"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/shared"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/stats"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/iwind/TeaGo/types"
	"github.com/pires/go-proxyproto"
	"net"
	"strings"
	"sync/atomic"
)

type TCPListener struct {
	BaseListener

	Listener net.Listener

	port int
}

func (this *TCPListener) Serve() error {
	var listener = this.Listener
	if this.Group.IsTLS() {
		listener = tls.NewListener(listener, this.buildTLSConfig())
	}

	// 获取分组端口
	var groupAddr = this.Group.Addr()
	var portIndex = strings.LastIndex(groupAddr, ":")
	if portIndex >= 0 {
		var port = groupAddr[portIndex+1:]
		this.port = types.Int(port)
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
				remotelogs.Error("TCP_LISTENER", err.Error())
			}
			atomic.AddInt64(&this.countActiveConnections, -1)
		}(conn)
	}

	return nil
}

func (this *TCPListener) Reload(group *serverconfigs.ServerAddressGroup) {
	this.Group = group
	this.Reset()
}

func (this *TCPListener) handleConn(conn net.Conn) error {
	var server = this.Group.FirstServer()
	if server == nil {
		return errors.New("no server available")
	}
	if server.ReverseProxy == nil {
		return errors.New("no ReverseProxy configured for the server")
	}

	// 绑定连接和服务
	clientConn, ok := conn.(ClientConnInterface)
	if ok {
		var goNext = clientConn.SetServerId(server.Id)
		if !goNext {
			return nil
		}
		clientConn.SetUserId(server.UserId)

		var userPlanId int64
		if server.UserPlan != nil && server.UserPlan.Id > 0 {
			userPlanId = server.UserPlan.Id
		}
		clientConn.SetUserPlanId(userPlanId)
	} else {
		tlsConn, ok := conn.(*tls.Conn)
		if ok {
			var internalConn = tlsConn.NetConn()
			if internalConn != nil {
				clientConn, ok = internalConn.(ClientConnInterface)
				if ok {
					var goNext = clientConn.SetServerId(server.Id)
					if !goNext {
						return nil
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
	}

	// 是否已达到流量限制
	if this.reachedTrafficLimit() || (server.UserId > 0 && !SharedUserManager.CheckUserServersIsEnabled(server.UserId)) {
		// 关闭连接
		tcpConn, ok := conn.(LingerConn)
		if ok {
			_ = tcpConn.SetLinger(0)
		}
		_ = conn.Close()

		// TODO 使用系统防火墙drop当前端口的数据包一段时间（1分钟）
		// 不能使用阻止IP的方法，因为边缘节点只上有可能还有别的代理服务

		return nil
	}

	// 记录域名排行
	tlsConn, ok := conn.(*tls.Conn)
	var recordStat = false
	var serverName = ""
	if ok {
		serverName = tlsConn.ConnectionState().ServerName
		if len(serverName) > 0 {
			// 统计
			stats.SharedTrafficStatManager.Add(server.UserId, server.Id, serverName, 0, 0, 1, 0, 0, 0, server.ShouldCheckTrafficLimit(), server.PlanId())
			recordStat = true
		}
	}

	// 统计
	if !recordStat {
		stats.SharedTrafficStatManager.Add(server.UserId, server.Id, "", 0, 0, 1, 0, 0, 0, server.ShouldCheckTrafficLimit(), server.PlanId())
	}

	originConn, err := this.connectOrigin(server.Id, serverName, server.ReverseProxy, conn.RemoteAddr().String())
	if err != nil {
		_ = conn.Close()
		return err
	}

	var closer = func() {
		_ = conn.Close()
		_ = originConn.Close()
	}

	// PROXY Protocol
	if server.ReverseProxy != nil &&
		server.ReverseProxy.ProxyProtocol != nil &&
		server.ReverseProxy.ProxyProtocol.IsOn &&
		(server.ReverseProxy.ProxyProtocol.Version == serverconfigs.ProxyProtocolVersion1 || server.ReverseProxy.ProxyProtocol.Version == serverconfigs.ProxyProtocolVersion2) {
		var remoteAddr = conn.RemoteAddr()
		var transportProtocol = proxyproto.TCPv4
		if strings.Contains(remoteAddr.String(), "[") {
			transportProtocol = proxyproto.TCPv6
		}
		var header = proxyproto.Header{
			Version:           byte(server.ReverseProxy.ProxyProtocol.Version),
			Command:           proxyproto.PROXY,
			TransportProtocol: transportProtocol,
			SourceAddr:        remoteAddr,
			DestinationAddr:   conn.LocalAddr(),
		}
		_, err = header.WriteTo(originConn)
		if err != nil {
			closer()
			return err
		}
	}

	// 从源站读取
	goman.New(func() {
		var originBuffer = utils.BytePool16k.Get()
		defer func() {
			utils.BytePool16k.Put(originBuffer)
		}()
		for {
			n, err := originConn.Read(originBuffer)
			if n > 0 {
				_, err = conn.Write(originBuffer[:n])
				if err != nil {
					closer()
					break
				}

				// 记录流量
				if server != nil {
					stats.SharedTrafficStatManager.Add(server.UserId, server.Id, "", int64(n), 0, 0, 0, 0, 0, server.ShouldCheckTrafficLimit(), server.PlanId())
				}
			}
			if err != nil {
				closer()
				break
			}
		}
	})

	// 从客户端读取
	var clientBuffer = utils.BytePool16k.Get()
	defer func() {
		utils.BytePool16k.Put(clientBuffer)
	}()
	for {
		// 是否已达到流量限制
		if this.reachedTrafficLimit() {
			closer()
			return nil
		}

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

// 连接源站
func (this *TCPListener) connectOrigin(serverId int64, requestHost string, reverseProxy *serverconfigs.ReverseProxyConfig, remoteAddr string) (conn net.Conn, err error) {
	if reverseProxy == nil {
		return nil, errors.New("no reverse proxy config")
	}

	var requestCall = shared.NewRequestCall()
	requestCall.Domain = requestHost

	var retries = 3
	var addr string

	var failedOriginIds []int64

	for i := 0; i < retries; i++ {
		var origin *serverconfigs.OriginConfig
		if len(failedOriginIds) > 0 {
			origin = reverseProxy.AnyOrigin(requestCall, failedOriginIds)
		}
		if origin == nil {
			origin = reverseProxy.NextOrigin(requestCall)
		}
		if origin == nil {
			continue
		}

		// 回源主机名
		if len(origin.RequestHost) > 0 {
			requestHost = origin.RequestHost
		} else if len(reverseProxy.RequestHost) > 0 {
			requestHost = reverseProxy.RequestHost
		}

		conn, addr, err = OriginConnect(origin, this.port, remoteAddr, requestHost)
		if err != nil {
			failedOriginIds = append(failedOriginIds, origin.Id)

			remotelogs.ServerError(serverId, "TCP_LISTENER", "unable to connect origin server: "+addr+": "+err.Error(), "", nil)

			SharedOriginStateManager.Fail(origin, requestHost, reverseProxy, func() {
				reverseProxy.ResetScheduling()
			})

			continue
		} else {
			if !origin.IsOk {
				SharedOriginStateManager.Success(origin, func() {
					reverseProxy.ResetScheduling()
				})
			}

			return
		}
	}

	if err == nil {
		err = errors.New("server '" + types.String(serverId) + "': no available origin server can be used")
	}
	return
}

// 检查是否已经达到流量限制
func (this *TCPListener) reachedTrafficLimit() bool {
	var server = this.Group.FirstServer()
	if server == nil {
		return true
	}
	return server.TrafficLimitStatus != nil && server.TrafficLimitStatus.IsValid()
}
