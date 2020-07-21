package configs

type UDPProtocolConfig struct {
	IsOn   bool     `yaml:"isOn"`                 // 是否开启
	Listen []string `yaml:"listen" json:"listen"` // 监听地址
}

func (this *UDPProtocolConfig) Addresses() []string {
	result := []string{}
	for _, listen := range this.Listen {
		result = append(result, ProtocolUDP+"://"+listen)
	}
	return result
}
