package nodes

import (
	"crypto/tls"
	"errors"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"net"
	"strconv"
)

// OriginConnect 连接源站
func OriginConnect(origin *serverconfigs.OriginConfig, remoteAddr string, tlsHost string) (net.Conn, error) {
	if origin.Addr == nil {
		return nil, errors.New("origin server address should not be empty")
	}

	// 支持TOA的连接
	// 这个条件很重要，如果没有传递remoteAddr，表示不使用TOA
	if len(remoteAddr) > 0 {
		toaConfig := sharedTOAManager.Config()
		if toaConfig != nil && toaConfig.IsOn {
			retries := 3
			for i := 1; i <= retries; i++ {
				port := int(toaConfig.RandLocalPort())
				err := sharedTOAManager.SendMsg("add:" + strconv.Itoa(port) + ":" + remoteAddr)
				if err != nil {
					remotelogs.Error("TOA", "add failed: "+err.Error())
				} else {
					dialer := net.Dialer{
						Timeout: origin.ConnTimeoutDuration(),
						LocalAddr: &net.TCPAddr{
							Port: port,
						},
					}
					var conn net.Conn
					switch origin.Addr.Protocol {
					case "", serverconfigs.ProtocolTCP, serverconfigs.ProtocolHTTP:
						// TODO 支持TCP4/TCP6
						// TODO 支持指定特定网卡
						// TODO Addr支持端口范围，如果有多个端口时，随机一个端口使用
						conn, err = dialer.Dial("tcp", origin.Addr.Host+":"+origin.Addr.PortRange)
					case serverconfigs.ProtocolTLS, serverconfigs.ProtocolHTTPS:
						// TODO 支持TCP4/TCP6
						// TODO 支持指定特定网卡
						// TODO Addr支持端口范围，如果有多个端口时，随机一个端口使用

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

						conn, err = tls.DialWithDialer(&dialer, "tcp", origin.Addr.Host+":"+origin.Addr.PortRange, tlsConfig)
					}

					// TODO 需要在合适的时机删除TOA记录
					if err == nil || i == retries {
						return conn, err
					}
				}
			}
		}
	}

	switch origin.Addr.Protocol {
	case "", serverconfigs.ProtocolTCP, serverconfigs.ProtocolHTTP:
		// TODO 支持TCP4/TCP6
		// TODO 支持指定特定网卡
		// TODO Addr支持端口范围，如果有多个端口时，随机一个端口使用
		return net.DialTimeout("tcp", origin.Addr.Host+":"+origin.Addr.PortRange, origin.ConnTimeoutDuration())
	case serverconfigs.ProtocolTLS, serverconfigs.ProtocolHTTPS:
		// TODO 支持TCP4/TCP6
		// TODO 支持指定特定网卡
		// TODO Addr支持端口范围，如果有多个端口时，随机一个端口使用

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

		return tls.Dial("tcp", origin.Addr.Host+":"+origin.Addr.PortRange, tlsConfig)
	case serverconfigs.ProtocolUDP:
		addr, err := net.ResolveUDPAddr("udp", origin.Addr.Host+":"+origin.Addr.PortRange)
		if err != nil {
			return nil, err
		}
		return net.DialUDP("udp", nil, addr)
	}

	// TODO 支持从Unix、Pipe、HTTP、HTTPS中读取数据

	return nil, errors.New("invalid origin scheme '" + origin.Addr.Protocol.String() + "'")
}
