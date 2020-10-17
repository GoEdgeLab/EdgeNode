package configs

import (
	"github.com/go-yaml/yaml"
	"github.com/iwind/TeaGo/Tea"
	"io/ioutil"
)

// 节点API配置
type APIConfig struct {
	RPC struct {
		Endpoints []string `yaml:"endpoints"`
	} `yaml:"rpc"`
	NodeId string `yaml:"nodeId"`
	Secret string `yaml:"secret"`
}

func LoadAPIConfig() (*APIConfig, error) {
	data, err := ioutil.ReadFile(Tea.ConfigFile("api.yaml"))
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

// 保存到文件
func (this *APIConfig) WriteFile(path string) error {
	data, err := yaml.Marshal(this)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(path, data, 0666)
	return err
}
