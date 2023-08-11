package iplibrary

import (
	"errors"
	"fmt"
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/firewallconfigs"
	executils "github.com/TeaOSLab/EdgeNode/internal/utils/exec"
	"runtime"
	"time"
)

// FirewalldAction Firewalld动作管理
// 常用命令：
//   - 查询列表： firewall-cmd --list-all
//   - 添加IP：firewall-cmd --add-rich-rule="rule family='ipv4' source address='192.168.2.32' reject" --timeout=30s
//   - 删除IP：firewall-cmd --remove-rich-rule="rule family='ipv4' source address='192.168.2.32' reject" --timeout=30s
type FirewalldAction struct {
	BaseAction

	config *firewallconfigs.FirewallActionFirewalldConfig

	firewalldNotFound bool
}

func NewFirewalldAction() *FirewalldAction {
	return &FirewalldAction{}
}

func (this *FirewalldAction) Init(config *firewallconfigs.FirewallActionConfig) error {
	this.config = &firewallconfigs.FirewallActionFirewalldConfig{}
	err := this.convertParams(config.Params, this.config)
	if err != nil {
		return err
	}
	return nil
}

func (this *FirewalldAction) AddItem(listType IPListType, item *pb.IPItem) error {
	return this.runAction("addItem", listType, item)
}

func (this *FirewalldAction) DeleteItem(listType IPListType, item *pb.IPItem) error {
	return this.runAction("deleteItem", listType, item)
}

func (this *FirewalldAction) runAction(action string, listType IPListType, item *pb.IPItem) error {
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

func (this *FirewalldAction) runActionSingleIP(action string, listType IPListType, item *pb.IPItem) error {
	timestamp := time.Now().Unix()

	if item.ExpiredAt > 0 && timestamp > item.ExpiredAt {
		return nil
	}

	path := this.config.Path
	var err error
	if len(path) == 0 {
		path, err = executils.LookPath("firewall-cmd")
		if err != nil {
			if this.firewalldNotFound {
				return nil
			}
			this.firewalldNotFound = true
			return err
		}
	}
	if len(path) == 0 {
		return errors.New("can not find 'firewall-cmd'")
	}

	opt := ""
	switch action {
	case "addItem":
		opt = "--add-rich-rule"
	case "deleteItem":
		opt = "--remove-rich-rule"
	default:
		return errors.New("invalid action '" + action + "'")
	}
	opt += "=rule family='"
	switch item.Type {
	case "ipv4":
		opt += "ipv4"
	case "ipv6":
		opt += "ipv6"
	default:
		// 我们忽略不能识别的Family
		return nil
	}

	opt += "' source address='"
	if len(item.IpFrom) == 0 {
		return errors.New("invalid ip from")
	}
	opt += item.IpFrom + "' "

	switch listType {
	case IPListTypeWhite:
		opt += " accept"
	case IPListTypeBlack:
		opt += " reject"
	default:
		// 我们忽略不能识别的列表类型
		return nil
	}

	args := []string{opt}
	if action == "addItem" {
		if item.ExpiredAt > timestamp {
			args = append(args, "--timeout="+fmt.Sprintf("%d", item.ExpiredAt-timestamp)+"s")
		} else {
			// TODO 思考是否需要permanent，不然--reload之后会丢失
		}
	}

	if runtime.GOOS == "darwin" {
		// MAC OS直接返回
		return nil
	}
	cmd := executils.NewTimeoutCmd(30*time.Second, path, args...)
	cmd.WithStderr()
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("%w, output: %s", err, cmd.Stderr())
	}
	return nil
}
