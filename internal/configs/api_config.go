package configs

import (
	"github.com/iwind/TeaGo/Tea"
	"gopkg.in/yaml.v3"
	"os"
)

// APIConfig 节点API配置
type APIConfig struct {
	RPC struct {
		Endpoints     []string `yaml:"endpoints"`
		DisableUpdate bool     `yaml:"disableUpdate"`
	} `yaml:"rpc"`
	NodeId string `yaml:"nodeId"`
	Secret string `yaml:"secret"`
}

func LoadAPIConfig() (*APIConfig, error) {
	data, err := os.ReadFile(Tea.ConfigFile("api.yaml"))
	if err != nil {
		return nil, err
	}

	config := &APIConfig{}
	err = yaml.Unmarshal(data, config)
	if err != nil {
		return nil, err
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
