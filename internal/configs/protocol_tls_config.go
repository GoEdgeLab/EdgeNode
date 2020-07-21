package configs

type TLSProtocolConfig struct {
	IsOn      bool     `yaml:"isOn"`                 // 是否开启
	IPVersion string   `yaml:"ipVersion"`            // 4, 6
	Listen    []string `yaml:"listen" json:"listen"` // 监听地址
}

func (this *TLSProtocolConfig) Addresses() []string {
	result := []string{}
	for _, listen := range this.Listen {
		switch this.IPVersion {
		case IPv4:
			result = append(result, ProtocolTLS4+"://"+listen)
		case IPv6:
			result = append(result, ProtocolTLS6+"://"+listen)
		default:
			result = append(result, ProtocolTLS+"://"+listen)
		}
	}
	return result
}
