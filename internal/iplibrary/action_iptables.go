package iplibrary

import (
	"fmt"
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/firewallconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	executils "github.com/TeaOSLab/EdgeNode/internal/utils/exec"
	"runtime"
	"strings"
	"time"
)

// IPTablesAction IPTables动作
// 相关命令：
//
//	iptables -A INPUT -s "192.168.2.32" -j ACCEPT
//	iptables -A INPUT -s "192.168.2.32" -j REJECT
//	iptables -D INPUT ...
//	iptables -F INPUT
type IPTablesAction struct {
	BaseAction

	config *firewallconfigs.FirewallActionIPTablesConfig

	iptablesNotFound bool
}

func NewIPTablesAction() *IPTablesAction {
	return &IPTablesAction{}
}

func (this *IPTablesAction) Init(config *firewallconfigs.FirewallActionConfig) error {
	this.config = &firewallconfigs.FirewallActionIPTablesConfig{}
	err := this.convertParams(config.Params, this.config)
	if err != nil {
		return err
	}
	return nil
}

func (this *IPTablesAction) AddItem(listType IPListType, item *pb.IPItem) error {
	return this.runAction("addItem", listType, item)
}

func (this *IPTablesAction) DeleteItem(listType IPListType, item *pb.IPItem) error {
	return this.runAction("deleteItem", listType, item)
}

func (this *IPTablesAction) runAction(action string, listType IPListType, item *pb.IPItem) error {
	if item.Type == "all" {
		return nil
	}
	if len(item.IpTo) == 0 {
		return this.runActionSingleIP(action, listType, item)
	}
	cidrList, err := iPv4RangeToCIDRRange(item.IpFrom, item.IpTo)
	if err != nil {
		// 不合法的范围不予处理即可
		return nil
	}
	if len(cidrList) == 0 {
		return nil
	}
	for _, cidr := range cidrList {
		item.IpFrom = cidr
		item.IpTo = ""
		err := this.runActionSingleIP(action, listType, item)
		if err != nil {
			return err
		}
	}
	return nil
}

func (this *IPTablesAction) runActionSingleIP(action string, listType IPListType, item *pb.IPItem) error {
	// 暂时不支持ipv6
	// TODO 将来支持ipv6
	if utils.IsIPv6(item.IpFrom) {
		return nil
	}

	if item.Type == "all" {
		return nil
	}
	var path = this.config.Path
	var err error
	if len(path) == 0 {
		path, err = executils.LookPath("iptables")
		if err != nil {
			if this.iptablesNotFound {
				return nil
			}
			this.iptablesNotFound = true
			return err
		}
		this.config.Path = path
	}
	iptablesAction := ""
	switch action {
	case "addItem":
		iptablesAction = "-A"
	case "deleteItem":
		iptablesAction = "-D"
	default:
		return nil
	}
	args := []string{iptablesAction, "INPUT", "-s", item.IpFrom, "-j"}
	switch listType {
	case IPListTypeWhite:
		args = append(args, "ACCEPT")
	case IPListTypeBlack:
		args = append(args, "REJECT")
	default:
		return nil
	}

	if runtime.GOOS == "darwin" {
		// MAC OS直接返回
		return nil
	}

	var cmd = executils.NewTimeoutCmd(30*time.Second, path, args...)
	cmd.WithStderr()
	err = cmd.Run()
	if err != nil {
		var output = cmd.Stderr()
		if strings.Contains(output, "No chain/target/match") {
			err = nil
		} else {
			return fmt.Errorf("%w, output: %s", err, output)
		}
	}
	return nil
}
