package configs

type WebConfig struct {
	Locations   []*LocationConfig  `yaml:"locations"`   // 路径规则

	// 本地静态资源配置
	Root string `yaml:"root" json:"root"` // 资源根目录
}
