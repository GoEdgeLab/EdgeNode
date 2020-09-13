package nodes

import "github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"

// 域名和服务映射
type NamedServer struct {
	Name   string                      // 匹配后的域名
	Server *serverconfigs.ServerConfig // 匹配后的服务配置
}
