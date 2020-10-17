package configs

// 集群配置
type ClusterConfig struct {
	RPC struct {
		Endpoints []string `yaml:"endpoints"`
	} `yaml:"rpc"`
	ClusterId string `yaml:"clusterId"`
	Secret string `yaml:"secret"`
}
