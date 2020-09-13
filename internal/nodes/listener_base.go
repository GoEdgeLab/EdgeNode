package nodes

import (
	"crypto/tls"
	"errors"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/sslconfigs"
	http2 "golang.org/x/net/http2"
	"sync"
)

type BaseListener struct {
	serversLocker      sync.RWMutex
	namedServersLocker sync.RWMutex
	namedServers       map[string]*NamedServer // 域名 => server
}

// 初始化
func (this *BaseListener) Init() {
	this.namedServers = map[string]*NamedServer{}
}

// 构造TLS配置
func (this *BaseListener) buildTLSConfig(group *serverconfigs.ServerGroup) *tls.Config {
	return &tls.Config{
		Certificates: nil,
		GetConfigForClient: func(info *tls.ClientHelloInfo) (config *tls.Config, e error) {
			ssl, _, err := this.matchSSL(group, info.ServerName)
			if err != nil {
				return nil, err
			}

			cipherSuites := ssl.TLSCipherSuites()
			if len(cipherSuites) == 0 {
				cipherSuites = nil
			}

			nextProto := []string{}
			if !ssl.HTTP2Disabled {
				nextProto = []string{http2.NextProtoTLS}
			}
			return &tls.Config{
				Certificates: nil,
				MinVersion:   ssl.TLSMinVersion(),
				CipherSuites: cipherSuites,
				GetCertificate: func(info *tls.ClientHelloInfo) (certificate *tls.Certificate, e error) {
					_, cert, err := this.matchSSL(group, info.ServerName)
					if err != nil {
						return nil, err
					}
					if cert == nil {
						return nil, errors.New("[proxy]no certs found for '" + info.ServerName + "'")
					}
					return cert, nil
				},
				ClientAuth: sslconfigs.GoSSLClientAuthType(ssl.ClientAuthType),
				ClientCAs:  ssl.CAPool(),

				NextProtos: nextProto,
			}, nil
		},
		GetCertificate: func(info *tls.ClientHelloInfo) (certificate *tls.Certificate, e error) {
			_, cert, err := this.matchSSL(group, info.ServerName)
			if err != nil {
				return nil, err
			}
			if cert == nil {
				return nil, errors.New("[proxy]no certs found for '" + info.ServerName + "'")
			}
			return cert, nil
		},
	}
}

// 根据域名匹配证书
func (this *BaseListener) matchSSL(group *serverconfigs.ServerGroup, domain string) (*sslconfigs.SSLConfig, *tls.Certificate, error) {
	this.serversLocker.RLock()
	defer this.serversLocker.RUnlock()

	// 如果域名为空，则取第一个
	// 通常域名为空是因为是直接通过IP访问的
	if len(domain) == 0 {
		if serverconfigs.SharedGlobalConfig().HTTPAll.MatchDomainStrictly {
			return nil, nil, errors.New("no tls server name matched")
		}

		firstServer := group.FirstServer()
		if firstServer == nil {
			return nil, nil, errors.New("no server available")
		}
		sslConfig := firstServer.SSLConfig()

		if sslConfig != nil {
			return sslConfig, sslConfig.FirstCert(), nil

		}
		return nil, nil, errors.New("no tls server name found")
	}

	// 通过代理服务域名配置匹配
	server, _ := this.findNamedServer(group, domain)
	if server == nil || server.SSLConfig() == nil || !server.SSLConfig().IsOn {
		// 搜索所有的Server，通过SSL证书内容中的DNSName匹配
		for _, server := range group.Servers {
			if server.SSLConfig() == nil || !server.SSLConfig().IsOn {
				continue
			}
			cert, ok := server.SSLConfig().MatchDomain(domain)
			if ok {
				return server.SSLConfig(), cert, nil
			}
		}

		return nil, nil, errors.New("[proxy]no server found for '" + domain + "'")
	}

	// 证书是否匹配
	sslConfig := server.SSLConfig()
	cert, ok := sslConfig.MatchDomain(domain)
	if ok {
		return sslConfig, cert, nil
	}

	return sslConfig, sslConfig.FirstCert(), nil
}

// 根据域名来查找匹配的域名
func (this *BaseListener) findNamedServer(group *serverconfigs.ServerGroup, name string) (serverConfig *serverconfigs.ServerConfig, serverName string) {
	// 读取缓存
	this.namedServersLocker.RLock()
	namedServer, found := this.namedServers[name]
	if found {
		this.namedServersLocker.RUnlock()
		return namedServer.Server, namedServer.Name
	}
	this.namedServersLocker.RUnlock()

	this.serversLocker.RLock()
	defer this.serversLocker.RUnlock()

	currentServers := group.Servers
	countServers := len(currentServers)
	if countServers == 0 {
		return nil, ""
	}

	// 只记录N个记录，防止内存耗尽
	maxNamedServers := 10240

	// 是否严格匹配域名
	matchDomainStrictly := serverconfigs.SharedGlobalConfig().HTTPAll.MatchDomainStrictly

	// 如果只有一个server，则默认为这个
	if countServers == 1 && !matchDomainStrictly {
		return currentServers[0], name
	}

	// 精确查找
	for _, server := range currentServers {
		if server.MatchNameStrictly(name) {
			this.namedServersLocker.Lock()
			if len(this.namedServers) < maxNamedServers {
				this.namedServers[name] = &NamedServer{
					Name:   name,
					Server: server,
				}
			}
			this.namedServersLocker.Unlock()
			return server, name
		}
	}

	// 模糊查找
	for _, server := range currentServers {
		if matched := server.MatchName(name); matched {
			this.namedServersLocker.Lock()
			if len(this.namedServers) < maxNamedServers {
				this.namedServers[name] = &NamedServer{
					Name:   name,
					Server: server,
				}
			}
			this.namedServersLocker.Unlock()
			return server, name
		}
	}

	// 找不到而且域名严格匹配模式下不返回Server
	if matchDomainStrictly {
		return nil, name
	}

	// 如果没有找到，则匹配到第一个
	return currentServers[0], name
}
