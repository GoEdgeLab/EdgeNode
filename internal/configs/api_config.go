package configs

import (
	"errors"
	"github.com/iwind/TeaGo/Tea"
	"gopkg.in/yaml.v3"
	"os"
)

// APIConfig 节点API配置
type APIConfig struct {
	RPC struct {
		Endpoints     []string `yaml:"endpoints" json:"endpoints"`
		DisableUpdate bool     `yaml:"disableUpdate" json:"disableUpdate"`
	} `yaml:"rpc" json:"rpc"`
	NodeId string `yaml:"nodeId" json:"nodeId"`
	Secret string `yaml:"secret" json:"secret"`
}

func NewAPIConfig() *APIConfig {
	return &APIConfig{}
}

func (this *APIConfig) Init() error {
	if len(this.RPC.Endpoints) == 0 {
		return errors.New("no valid 'rpc.endpoints'")
	}
	if len(this.NodeId) == 0 {
		return errors.New("'nodeId' required")
	}
	if len(this.Secret) == 0 {
		return errors.New("'secret' required")
	}
	return nil
}

func LoadAPIConfig() (*APIConfig, error) {
	data, err := os.ReadFile(Tea.ConfigFile("api.yaml"))
	if err != nil {
		return nil, err
	}

	var config = &APIConfig{}
	err = yaml.Unmarshal(data, config)
	if err != nil {
		return nil, err
	}

	err = config.Init()
	if err != nil {
		return nil, errors.New("init error: " + err.Error())
	}

	return config, nil
}

// WriteFile 保存到文件
func (this *APIConfig) WriteFile(path string) error {
	data, err := yaml.Marshal(this)
	if err != nil {
		return err
	}
	err = os.WriteFile(path, data, 0666)
	return err
}
