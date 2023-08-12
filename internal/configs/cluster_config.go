package configs

import (
	"github.com/iwind/TeaGo/Tea"
	"gopkg.in/yaml.v3"
	"os"
)

// ClusterConfig 集群配置
type ClusterConfig struct {
	OldRPC struct {
		Endpoints     []string `yaml:"endpoints" json:"endpoints"`
		DisableUpdate bool     `yaml:"disableUpdate" json:"disableUpdate"`
	} `yaml:"rpc,omitempty" json:"rpc"`

	RPCEndpoints     []string `yaml:"rpc.endpoints,flow" json:"rpc.endpoints"`
	RPCDisableUpdate bool     `yaml:"rpc.disableUpdate" json:"rpc.disableUpdate"`

	ClusterId string `yaml:"clusterId" json:"clusterId"`
	Secret    string `yaml:"secret" json:"secret"`
}

func (this *ClusterConfig) Init() error {
	// compatible with old
	if len(this.RPCEndpoints) == 0 && len(this.OldRPC.Endpoints) > 0 {
		this.RPCEndpoints = this.OldRPC.Endpoints
		this.RPCDisableUpdate = this.OldRPC.DisableUpdate
	}

	return nil
}

func LoadClusterConfig() (*ClusterConfig, error) {
	for _, filename := range []string{"api_cluster.yaml", "cluster.yaml"} {
		data, err := os.ReadFile(Tea.ConfigFile(filename))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}

		var config = &ClusterConfig{}
		err = yaml.Unmarshal(data, config)
		if err != nil {
			return config, err
		}

		err = config.Init()
		if err != nil {
			return nil, err
		}

		return config, nil
	}
	return nil, os.ErrNotExist
}
