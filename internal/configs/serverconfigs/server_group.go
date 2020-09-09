package serverconfigs

import "strings"

type ServerGroup struct {
	fullAddr string
	Servers  []*ServerConfig
}

func NewServerGroup(fullAddr string) *ServerGroup {
	return &ServerGroup{fullAddr: fullAddr}
}

// 添加服务
func (this *ServerGroup) Add(server *ServerConfig) {
	this.Servers = append(this.Servers, server)
}

// 获取完整的地址
func (this *ServerGroup) FullAddr() string {
	return this.fullAddr
}

// 获取当前分组的协议
func (this *ServerGroup) Protocol() Protocol {
	for _, p := range AllProtocols() {
		if strings.HasPrefix(this.fullAddr, p+":") {
			return p
		}
	}
	return ProtocolHTTP
}

// 获取当前分组的地址
func (this *ServerGroup) Addr() string {
	protocol := this.Protocol()
	if protocol == ProtocolUnix {
		return strings.TrimPrefix(this.fullAddr, protocol+":")
	}
	return strings.TrimPrefix(this.fullAddr, protocol+"://")
}

// 判断当前分组是否为HTTP
func (this *ServerGroup) IsHTTP() bool {
	p := this.Protocol()
	return p == ProtocolHTTP || p == ProtocolHTTP4 || p == ProtocolHTTP6
}

// 判断当前分组是否为HTTPS
func (this *ServerGroup) IsHTTPS() bool {
	p := this.Protocol()
	return p == ProtocolHTTPS || p == ProtocolHTTPS4 || p == ProtocolHTTPS6
}

// 判断当前分组是否为TCP
func (this *ServerGroup) IsTCP() bool {
	p := this.Protocol()
	return p == ProtocolTCP || p == ProtocolTCP4 || p == ProtocolTCP6
}

// 判断当前分组是否为TLS
func (this *ServerGroup) IsTLS() bool {
	p := this.Protocol()
	return p == ProtocolTLS || p == ProtocolTLS4 || p == ProtocolTLS6
}

// 判断当前分组是否为Unix
func (this *ServerGroup) IsUnix() bool {
	p := this.Protocol()
	return p == ProtocolUnix
}

// 判断当前分组是否为UDP
func (this *ServerGroup) IsUDP() bool {
	p := this.Protocol()
	return p == ProtocolUDP
}

// 获取第一个Server
func (this *ServerGroup) FirstServer() *ServerConfig {
	if len(this.Servers) > 0 {
		return this.Servers[0]
	}
	return nil
}
