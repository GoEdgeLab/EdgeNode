package configs

// ClusterConfig 集群配置
type ClusterConfig struct {
	RPC struct {
		Endpoints     []string `yaml:"endpoints" json:"endpoints"`
		DisableUpdate bool     `yaml:"disableUpdate" json:"disableUpdate"`
	} `yaml:"rpc" json:"rpc"`
	ClusterId string `yaml:"clusterId" json:"clusterId"`
	Secret    string `yaml:"secret" json:"secret"`
}
