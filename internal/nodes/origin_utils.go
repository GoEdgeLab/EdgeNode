package nodes

import (
	"crypto/tls"
	"errors"
	"github.com/TeaOSLab/EdgeCommon/pkg/configutils"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/iwind/TeaGo/types"
	"net"
	"strconv"
)

// OriginConnect 连接源站
func OriginConnect(origin *serverconfigs.OriginConfig, serverPort int, remoteAddr string, tlsHost string) (originConn net.Conn, originAddr string, err error) {
	if origin.Addr == nil {
		return nil, "", errors.New("origin server address should not be empty")
	}

	// 支持TOA的连接
	// 这个条件很重要，如果没有传递remoteAddr，表示不使用TOA
	if len(remoteAddr) > 0 {
		var toaConfig = sharedTOAManager.Config()
		if toaConfig != nil && toaConfig.IsOn {
			var retries = 3
			for i := 1; i <= retries; i++ {
				var port = int(toaConfig.RandLocalPort())
				err = sharedTOAManager.SendMsg("add:" + strconv.Itoa(port) + ":" + remoteAddr)
				if err != nil {
					remotelogs.Error("TOA", "add failed: "+err.Error())
				} else {
					var dialer = net.Dialer{
						Timeout: origin.ConnTimeoutDuration(),
						LocalAddr: &net.TCPAddr{
							Port: port,
						},
					}
					originAddr = origin.Addr.PickAddress()

					// 端口跟随
					if origin.FollowPort && serverPort > 0 {
						originAddr = configutils.QuoteIP(origin.Addr.Host) + ":" + types.String(serverPort)
					}

					var conn net.Conn
					switch origin.Addr.Protocol {
					case "", serverconfigs.ProtocolTCP, serverconfigs.ProtocolHTTP:
						// TODO 支持TCP4/TCP6
						// TODO 支持指定特定网卡
						conn, err = dialer.Dial("tcp", originAddr)
					case serverconfigs.ProtocolTLS, serverconfigs.ProtocolHTTPS:
						// TODO 支持TCP4/TCP6
						// TODO 支持指定特定网卡

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
						if len(tlsHost) > 0 {
							tlsConfig.ServerName = tlsHost
						}

						conn, err = tls.DialWithDialer(&dialer, "tcp", originAddr, tlsConfig)
					}

					// TODO 需要在合适的时机删除TOA记录
					if err == nil || i == retries {
						return conn, originAddr, err
					}
				}
			}
		}
	}

	originAddr = origin.Addr.PickAddress()

	// 端口跟随
	if origin.FollowPort && serverPort > 0 {
		originAddr = configutils.QuoteIP(origin.Addr.Host) + ":" + types.String(serverPort)
	}

	switch origin.Addr.Protocol {
	case "", serverconfigs.ProtocolTCP, serverconfigs.ProtocolHTTP:
		// TODO 支持TCP4/TCP6
		// TODO 支持指定特定网卡
		originConn, err = net.DialTimeout("tcp", originAddr, origin.ConnTimeoutDuration())
		return originConn, originAddr, err
	case serverconfigs.ProtocolTLS, serverconfigs.ProtocolHTTPS:
		// TODO 支持TCP4/TCP6
		// TODO 支持指定特定网卡

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
		if len(tlsHost) > 0 {
			tlsConfig.ServerName = tlsHost
		}

		originConn, err = tls.Dial("tcp", originAddr, tlsConfig)
		return originConn, originAddr, err
	case serverconfigs.ProtocolUDP:
		addr, err := net.ResolveUDPAddr("udp", originAddr)
		if err != nil {
			return nil, originAddr, err
		}
		originConn, err = net.DialUDP("udp", nil, addr)
		return originConn, originAddr, err
	}

	// TODO 支持从Unix、Pipe、HTTP、HTTPS中读取数据

	return nil, originAddr, errors.New("invalid origin scheme '" + origin.Addr.Protocol.String() + "'")
}
