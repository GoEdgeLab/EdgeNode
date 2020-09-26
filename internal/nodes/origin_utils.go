package nodes

import (
	"crypto/tls"
	"errors"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"net"
)

// 连接源站
func OriginConnect(origin *serverconfigs.OriginConfig) (net.Conn, error) {
	if origin.Addr == nil {
		return nil, errors.New("origin server address should not be empty")
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
		return tls.Dial("tcp", origin.Addr.Host+":"+origin.Addr.PortRange, &tls.Config{})
	}

	// TODO 支持从Unix、Pipe、HTTP、HTTPS中读取数据

	return nil, errors.New("invalid scheme '" + origin.Addr.Protocol.String() + "'")
}
