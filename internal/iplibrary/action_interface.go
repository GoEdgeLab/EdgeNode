package iplibrary

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/firewallconfigs"
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
}
