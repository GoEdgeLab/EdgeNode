package configs

type TCPProtocolConfig struct {
	IsOn      bool      `yaml:"isOn"`                 // 是否开启
	IPVersion IPVersion `yaml:"ipVersion"`            // 4, 6
	Listen    []string  `yaml:"listen" json:"listen"` // 监听地址
}

func (this *TCPProtocolConfig) Addresses() []string {
	result := []string{}
	for _, listen := range this.Listen {
		switch this.IPVersion {
		case IPv4:
			result = append(result, ProtocolTCP4+"://"+listen)
		case IPv6:
			result = append(result, ProtocolTCP6+"://"+listen)
		default:
			result = append(result, ProtocolTCP+"://"+listen)
		}
	}
	return result
}
