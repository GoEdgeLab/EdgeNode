package iplibrary

import (
	"bytes"
	"errors"
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/firewallconfigs"
	"os/exec"
	"runtime"
)

// IPTables动作
// 相关命令：
//   iptables -A INPUT -s "192.168.2.32" -j ACCEPT
//   iptables -A INPUT -s "192.168.2.32" -j REJECT
//   iptables -D ...
type IPTablesAction struct {
	BaseAction

	config *firewallconfigs.FirewallActionIPTablesConfig
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
	if item.Type == "all" {
		return nil
	}
	path := this.config.Path
	var err error
	if len(path) == 0 {
		path, err = exec.LookPath("iptables")
		if err != nil {
			return err
		}
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

	cmd := exec.Command(path, args...)
	stderr := bytes.NewBuffer([]byte{})
	cmd.Stderr = stderr
	err = cmd.Run()
	if err != nil {
		output := stderr.Bytes()
		if bytes.Contains(output, []byte("No chain/target/match")) {
			err = nil
		} else {
			return errors.New(err.Error() + ", output: " + string(output))
		}
	}
	return nil
}
