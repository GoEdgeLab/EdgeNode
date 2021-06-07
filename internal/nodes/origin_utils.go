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
func OriginConnect(origin *serverconfigs.OriginConfig, remoteAddr string) (net.Conn, error) {
	if origin.Addr == nil {
		return nil, errors.New("origin server address should not be empty")
	}

	// 支持TOA的连接
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
					// TODO 支持使用证书
					conn, err = tls.DialWithDialer(&dialer, "tcp", origin.Addr.Host+":"+origin.Addr.PortRange, &tls.Config{
						InsecureSkipVerify: true,
					})
				}

				// TODO 需要在合适的时机删除TOA记录
				if err == nil || i == retries {
					return conn, err
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
		// TODO 支持使用证书
		return tls.Dial("tcp", origin.Addr.Host+":"+origin.Addr.PortRange, &tls.Config{
			InsecureSkipVerify: true,
		})
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
