package configs

// ClusterConfig 集群配置
type ClusterConfig struct {
	RPC struct {
		Endpoints     []string `yaml:"endpoints"`
		DisableUpdate bool     `yaml:"disableUpdate"`
	} `yaml:"rpc"`
	ClusterId string `yaml:"clusterId"`
	Secret    string `yaml:"secret"`
}
