package serverconfigs

import (
	"github.com/TeaOSLab/EdgeNode/internal/configs/serverconfigs/configutils"
	"github.com/TeaOSLab/EdgeNode/internal/configs/serverconfigs/shared"
)

var globalConfig *GlobalConfig = nil
var globalConfigFile = "global.yaml"

// 全局设置
type GlobalConfig struct {
	HTTPAll struct {
		MatchDomainStrictly bool `yaml:"matchDomainStrictly"`
	} `yaml:"httpAll"`
	HTTP   struct{} `yaml:"http"`
	HTTPS  struct{} `yaml:"https"`
	TCPAll struct{} `yaml:"tcpAll"`
	TCP    struct{} `yaml:"tcp"`
	TLS    struct{} `yaml:"tls"`
	Unix   struct{} `yaml:"unix"`
	UDP    struct{} `yaml:"udp"`
}

func SharedGlobalConfig() *GlobalConfig {
	shared.Locker.Lock()
	defer shared.Locker.Unlock()

	if globalConfig != nil {
		return globalConfig
	}

	err := configutils.UnmarshalYamlFile(globalConfigFile, globalConfig)
	if err != nil {
		configutils.LogError("[SharedGlobalConfig]" + err.Error())
		globalConfig = &GlobalConfig{}
	}
	return globalConfig
}

func (this *GlobalConfig) Init() error {
	return nil
}
