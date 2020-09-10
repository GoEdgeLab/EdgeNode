package serverconfigs

import "github.com/TeaOSLab/EdgeNode/internal/configs/serverconfigs/sslconfigs"

type TLSProtocolConfig struct {
	BaseProtocol `yaml:",inline"`

	SSL *sslconfigs.SSLConfig `yaml:"ssl" json:"ssl"`
}

// 初始化
func (this *TLSProtocolConfig) Init() error {
	err := this.InitBase()
	if err != nil {
		return err
	}

	return nil
}
