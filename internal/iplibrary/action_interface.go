package iplibrary

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/firewallconfigs"
	"net/http"
)

type ActionInterface interface {
	// 初始化
	Init(config *firewallconfigs.FirewallActionConfig) error

	// 添加
	AddItem(listType IPListType, item *pb.IPItem) error

	// 删除
	DeleteItem(listType IPListType, item *pb.IPItem) error

	// 关闭
	Close() error

	// 处理HTTP请求
	DoHTTP(req *http.Request, resp http.ResponseWriter) (goNext bool, err error)
}
