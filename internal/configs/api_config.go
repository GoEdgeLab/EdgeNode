package configs

import (
	"errors"
	"github.com/iwind/TeaGo/Tea"
	"gopkg.in/yaml.v3"
	"os"
)

const ConfigFileName = "api_node.yaml"
const oldConfigFileName = "api.yaml"

type APIConfig struct {
	OldRPC struct {
		Endpoints     []string `yaml:"endpoints" json:"endpoints"`
		DisableUpdate bool     `yaml:"disableUpdate" json:"disableUpdate"`
	} `yaml:"rpc,omitempty" json:"rpc"`

	RPCEndpoints     []string `yaml:"rpc.endpoints,flow" json:"rpc.endpoints"`
	RPCDisableUpdate bool     `yaml:"rpc.disableUpdate" json:"rpc.disableUpdate"`
	NodeId           string   `yaml:"nodeId" json:"nodeId"`
	Secret           string   `yaml:"secret" json:"secret"`
}

func NewAPIConfig() *APIConfig {
	return &APIConfig{}
}

func (this *APIConfig) Init() error {
	// compatible with old
	if len(this.RPCEndpoints) == 0 && len(this.OldRPC.Endpoints) > 0 {
		this.RPCEndpoints = this.OldRPC.Endpoints
		this.RPCDisableUpdate = this.OldRPC.DisableUpdate
	}

	if len(this.RPCEndpoints) == 0 {
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
	for _, filename := range []string{ConfigFileName, oldConfigFileName} {
		data, err := os.ReadFile(Tea.ConfigFile(filename))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
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

		// 自动生成新的配置文件
		if filename == oldConfigFileName {
			config.OldRPC.Endpoints = nil
			_ = config.WriteFile(Tea.ConfigFile(ConfigFileName))
		}

		return config, nil
	}
	return nil, errors.New("no config file '" + ConfigFileName + "' found")
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
