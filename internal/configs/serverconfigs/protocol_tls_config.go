package serverconfigs

import "github.com/TeaOSLab/EdgeNode/internal/configs/serverconfigs/sslconfigs"

type TLSProtocolConfig struct {
	BaseProtocol `yaml:",inline"`

	SSL *sslconfigs.SSLConfig `yaml:"ssl"`
}

func (this *TLSProtocolConfig) Init() error {
	err := this.InitBase()
	if err != nil {
		return err
	}

	return nil
}
