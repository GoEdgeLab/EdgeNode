package nodes

import (
	"crypto/tls"
	"errors"
	"github.com/TeaOSLab/EdgeCommon/pkg/configutils"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/sslconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/iwind/TeaGo/types"
	"net"
)

type BaseListener struct {
	Group *serverconfigs.ServerAddressGroup

	countActiveConnections int64 // 当前活跃的连接数
}

// Init 初始化
func (this *BaseListener) Init() {
}

// Reset 清除既有配置
func (this *BaseListener) Reset() {

}

// CountActiveConnections 获取当前活跃连接数
func (this *BaseListener) CountActiveConnections() int {
	return types.Int(this.countActiveConnections)
}

// 构造TLS配置
func (this *BaseListener) buildTLSConfig() *tls.Config {
	return &tls.Config{
		Certificates: nil,
		GetConfigForClient: func(clientInfo *tls.ClientHelloInfo) (config *tls.Config, e error) {
			// 指纹信息
			var fingerprint = this.calculateFingerprint(clientInfo)
			if len(fingerprint) > 0 && clientInfo.Conn != nil {
				clientConn, ok := clientInfo.Conn.(ClientConnInterface)
				if ok {
					clientConn.SetFingerprint(fingerprint)
				}
			}

			tlsPolicy, _, err := this.matchSSL(this.helloServerName(clientInfo))
			if err != nil {
				return nil, err
			}

			if tlsPolicy == nil {
				return nil, nil
			}

			tlsPolicy.CheckOCSP()

			return tlsPolicy.TLSConfig(), nil
		},
		GetCertificate: func(clientInfo *tls.ClientHelloInfo) (certificate *tls.Certificate, e error) {
			// 指纹信息
			var fingerprint = this.calculateFingerprint(clientInfo)
			if len(fingerprint) > 0 && clientInfo.Conn != nil {
				clientConn, ok := clientInfo.Conn.(ClientConnInterface)
				if ok {
					clientConn.SetFingerprint(fingerprint)
				}
			}

			tlsPolicy, cert, err := this.matchSSL(this.helloServerName(clientInfo))
			if err != nil {
				return nil, err
			}
			if cert == nil {
				return nil, errors.New("no ssl certs found for '" + clientInfo.ServerName + "'")
			}

			tlsPolicy.CheckOCSP()

			return cert, nil
		},
	}
}

// 根据域名匹配证书
func (this *BaseListener) matchSSL(domain string) (*sslconfigs.SSLPolicy, *tls.Certificate, error) {
	var group = this.Group

	if group == nil {
		return nil, nil, errors.New("no configure found")
	}

	var globalServerConfig *serverconfigs.GlobalServerConfig
	if sharedNodeConfig != nil {
		globalServerConfig = sharedNodeConfig.GlobalServerConfig
	}

	// 如果域名为空，则取第一个
	// 通常域名为空是因为是直接通过IP访问的
	if len(domain) == 0 {
		if group.IsHTTPS() && globalServerConfig != nil && globalServerConfig.HTTPAll.MatchDomainStrictly {
			return nil, nil, errors.New("no tls server name matched")
		}

		firstServer := group.FirstTLSServer()
		if firstServer == nil {
			return nil, nil, errors.New("no tls server available")
		}
		sslConfig := firstServer.SSLPolicy()

		if sslConfig != nil {
			return sslConfig, sslConfig.FirstCert(), nil

		}
		return nil, nil, errors.New("no tls server name found")
	}

	// 通过网站域名配置匹配
	server, _ := this.findNamedServer(domain)
	if server == nil {
		// 找不到或者此时的服务没有配置证书，需要搜索所有的Server，通过SSL证书内容中的DNSName匹配
		// 此功能仅为了兼容以往版本（v1.0.4），不应该作为常态启用
		if globalServerConfig != nil && globalServerConfig.HTTPAll.MatchCertFromAllServers {
			for _, searchingServer := range group.Servers() {
				if searchingServer.SSLPolicy() == nil || !searchingServer.SSLPolicy().IsOn {
					continue
				}
				cert, ok := searchingServer.SSLPolicy().MatchDomain(domain)
				if ok {
					return searchingServer.SSLPolicy(), cert, nil
				}
			}
		}

		return nil, nil, errors.New("no server found for '" + domain + "'")
	}
	if server.SSLPolicy() == nil || !server.SSLPolicy().IsOn {
		// 找不到或者此时的服务没有配置证书，需要搜索所有的Server，通过SSL证书内容中的DNSName匹配
		// 此功能仅为了兼容以往版本（v1.0.4），不应该作为常态启用
		if globalServerConfig != nil && globalServerConfig.HTTPAll.MatchCertFromAllServers {
			for _, searchingServer := range group.Servers() {
				if searchingServer.SSLPolicy() == nil || !searchingServer.SSLPolicy().IsOn {
					continue
				}
				cert, ok := searchingServer.SSLPolicy().MatchDomain(domain)
				if ok {
					return searchingServer.SSLPolicy(), cert, nil
				}
			}
		}

		return nil, nil, errors.New("no cert found for '" + domain + "'")
	}

	// 证书是否匹配
	var sslConfig = server.SSLPolicy()
	cert, ok := sslConfig.MatchDomain(domain)
	if ok {
		return sslConfig, cert, nil
	}

	if len(sslConfig.Certs) == 0 {
		remotelogs.ServerError(server.Id, "BASE_LISTENER", "no ssl certs found for '"+domain+"', server id: "+types.String(server.Id), "", nil)
	}

	return sslConfig, sslConfig.FirstCert(), nil
}

// 根据域名来查找匹配的域名
func (this *BaseListener) findNamedServer(name string) (serverConfig *serverconfigs.ServerConfig, serverName string) {
	serverConfig, serverName = this.findNamedServerMatched(name)
	if serverConfig != nil {
		return
	}

	var globalServerConfig = sharedNodeConfig.GlobalServerConfig
	var matchDomainStrictly = globalServerConfig != nil && globalServerConfig.HTTPAll.MatchDomainStrictly

	if globalServerConfig != nil &&
		len(globalServerConfig.HTTPAll.DefaultDomain) > 0 &&
		(!matchDomainStrictly || configutils.MatchDomains(globalServerConfig.HTTPAll.AllowMismatchDomains, name) || (globalServerConfig.HTTPAll.AllowNodeIP && utils.IsWildIP(name))) {
		if globalServerConfig.HTTPAll.AllowNodeIP &&
			globalServerConfig.HTTPAll.NodeIPShowPage &&
			utils.IsWildIP(name) {
			return
		} else {
			var defaultDomain = globalServerConfig.HTTPAll.DefaultDomain
			serverConfig, serverName = this.findNamedServerMatched(defaultDomain)
			if serverConfig != nil {
				return
			}
		}
	}

	if matchDomainStrictly && !configutils.MatchDomains(globalServerConfig.HTTPAll.AllowMismatchDomains, name) && (!globalServerConfig.HTTPAll.AllowNodeIP || !utils.IsWildIP(name)) {
		return
	}

	// 如果没有找到，则匹配到第一个
	var group = this.Group
	var currentServers = group.Servers()
	var countServers = len(currentServers)
	if countServers == 0 {
		return nil, ""
	}
	return currentServers[0], name
}

// 严格查找域名
func (this *BaseListener) findNamedServerMatched(name string) (serverConfig *serverconfigs.ServerConfig, serverName string) {
	var group = this.Group
	if group == nil {
		return nil, ""
	}

	server := group.MatchServerName(name)
	if server != nil {
		return server, name
	}

	// 是否严格匹配域名
	var matchDomainStrictly = sharedNodeConfig.GlobalServerConfig != nil && sharedNodeConfig.GlobalServerConfig.HTTPAll.MatchDomainStrictly

	// 如果只有一个server，则默认为这个
	var currentServers = group.Servers()
	var countServers = len(currentServers)
	if countServers == 1 && !matchDomainStrictly {
		return currentServers[0], name
	}

	return nil, name
}

// 从Hello信息中获取服务名称
func (this *BaseListener) helloServerName(clientInfo *tls.ClientHelloInfo) string {
	var serverName = clientInfo.ServerName
	if len(serverName) == 0 && clientInfo.Conn != nil {
		var localAddr = clientInfo.Conn.LocalAddr()
		if localAddr != nil {
			tcpAddr, ok := localAddr.(*net.TCPAddr)
			if ok {
				serverName = tcpAddr.IP.String()
			}
		}
	}
	return serverName
}
