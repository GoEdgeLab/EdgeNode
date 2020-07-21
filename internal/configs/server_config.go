package configs

type ServerConfig struct {
	Id          string             `yaml:"id"`          // ID
	IsOn        bool               `yaml:"isOn"`        // 是否开启
	Components  []*ComponentConfig `yaml:"components"`  // 组件
	Name        string             `yaml:"name"`        // 名称
	Description string             `yaml:"description"` // 描述
	ServerNames []string           `yaml:"serverNames"` // 域名

	// 协议
	HTTP  *HTTPProtocolConfig  `yaml:"http"`  // HTTP配置
	HTTPS *HTTPSProtocolConfig `yaml:"https"` // HTTPS配置
	TCP   *TCPProtocolConfig   `yaml:"tcp"`   // TCP配置
	TLS   *TLSProtocolConfig   `yaml:"tls"`   // TLS配置
	Unix  *UnixProtocolConfig  `yaml:"unix"`  // Unix配置
	UDP   *UDPProtocolConfig   `yaml:"udp"`   // UDP配置

	// Web配置
	Web *WebConfig `yaml:"web"`
}

func NewServerConfig() *ServerConfig {
	return &ServerConfig{}
}

func (this *ServerConfig) Init() error {
	return nil
}

func (this *ServerConfig) FullAddresses() []string {
	result := []Protocol{}
	if this.HTTP != nil && this.HTTP.IsOn {
		result = append(result, this.HTTP.Addresses()...)
	}
	if this.HTTPS != nil && this.HTTPS.IsOn {
		result = append(result, this.HTTPS.Addresses()...)
	}
	if this.TCP != nil && this.TCP.IsOn {
		result = append(result, this.TCP.Addresses()...)
	}
	if this.TLS != nil && this.TLS.IsOn {
		result = append(result, this.TLS.Addresses()...)
	}
	if this.Unix != nil && this.Unix.IsOn {
		result = append(result, this.Unix.Addresses()...)
	}
	if this.UDP != nil && this.UDP.IsOn {
		result = append(result, this.UDP.Addresses()...)
	}

	return result
}
