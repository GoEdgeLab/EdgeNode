package iplibrary

import (
	"errors"
	"fmt"
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/firewallconfigs"
	executils "github.com/TeaOSLab/EdgeNode/internal/utils/exec"
	"github.com/iwind/TeaGo/types"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// IPSetAction IPSet动作
// 相关命令：
//   - 利用Firewalld管理set：
//   - 添加：firewall-cmd --permanent --new-ipset=edge_ip_list --type=hash:ip --option="timeout=0"
//   - 删除：firewall-cmd --permanent --delete-ipset=edge_ip_list
//   - 重载：firewall-cmd --reload
//   - firewalld+ipset: firewall-cmd --permanent --add-rich-rule="rule source ipset='edge_ip_list' reject"
//   - 利用IPTables管理set：
//   - 添加：iptables -A INPUT -m set --match-set edge_ip_list src -j REJECT
//   - 添加Item：ipset add edge_ip_list 192.168.2.32 timeout 30
//   - 删除Item: ipset del edge_ip_list 192.168.2.32
//   - 创建set：ipset create edge_ip_list hash:ip timeout 0
//   - 查看统计：ipset -t list edge_black_list
//   - 删除set：ipset destroy edge_black_list
type IPSetAction struct {
	BaseAction

	config *firewallconfigs.FirewallActionIPSetConfig

	ipsetNotfound bool
}

func NewIPSetAction() *IPSetAction {
	return &IPSetAction{}
}

func (this *IPSetAction) Init(config *firewallconfigs.FirewallActionConfig) error {
	this.config = &firewallconfigs.FirewallActionIPSetConfig{}
	err := this.convertParams(config.Params, this.config)
	if err != nil {
		return err
	}

	if len(this.config.WhiteName) == 0 {
		return NewFataError("white list name should not be empty")
	}
	if len(this.config.BlackName) == 0 {
		return NewFataError("black list name should not be empty")
	}

	// 创建ipset
	{
		path, err := executils.LookPath("ipset")
		if err != nil {
			return err
		}

		// ipv4
		for _, listName := range []string{this.config.WhiteName, this.config.BlackName} {
			if len(listName) == 0 {
				continue
			}
			var cmd = executils.NewTimeoutCmd(30*time.Second, path, "create", listName, "hash:ip", "timeout", "0", "maxelem", "1000000")
			cmd.WithStderr()
			err := cmd.Run()
			if err != nil {
				var output = cmd.Stderr()
				if !strings.Contains(output, "already exists") {
					return fmt.Errorf("create ipset '%s': %w, output: %s", listName, err, output)
				} else {
					err = nil
				}
			}
		}

		// ipv6
		for _, listName := range []string{this.config.WhiteNameIPv6, this.config.BlackNameIPv6} {
			if len(listName) == 0 {
				continue
			}
			var cmd = executils.NewTimeoutCmd(30*time.Second, path, "create", listName, "hash:ip", "family", "inet6", "timeout", "0", "maxelem", "1000000")
			cmd.WithStderr()
			err := cmd.Run()
			if err != nil {
				var output = cmd.Stderr()
				if !strings.Contains(output, "already exists") {
					return fmt.Errorf("create ipset '%s': %w, output: %s", listName, err, output)
				} else {
					err = nil
				}
			}
		}
	}

	// firewalld
	if this.config.AutoAddToFirewalld {
		path, err := executils.LookPath("firewall-cmd")
		if err != nil {
			return err
		}

		// ipv4
		for _, listName := range []string{this.config.WhiteName, this.config.BlackName} {
			if len(listName) == 0 {
				continue
			}
			var cmd = executils.NewTimeoutCmd(30*time.Second, path, "--permanent", "--new-ipset="+listName, "--type=hash:ip", "--option=timeout=0", "--option=maxelem=1000000")
			cmd.WithStderr()
			err := cmd.Run()
			if err != nil {
				var output = cmd.Stderr()
				if strings.Contains(output, "NAME_CONFLICT") {
					err = nil
				} else {
					return fmt.Errorf("firewall-cmd add ipset '%s': %w, output: %s", listName, err, output)
				}
			}
		}

		// ipv6
		for _, listName := range []string{this.config.WhiteNameIPv6, this.config.BlackNameIPv6} {
			if len(listName) == 0 {
				continue
			}
			var cmd = executils.NewTimeoutCmd(30*time.Second, path, "--permanent", "--new-ipset="+listName, "--type=hash:ip", "--option=family=inet6", "--option=timeout=0", "--option=maxelem=1000000")
			cmd.WithStderr()
			err := cmd.Run()
			if err != nil {
				var output = cmd.Stderr()
				if strings.Contains(output, "NAME_CONFLICT") {
					err = nil
				} else {
					return fmt.Errorf("firewall-cmd add ipset '%s': %w, output: %s", listName, err, output)
				}
			}
		}

		// accept
		for _, listName := range []string{this.config.WhiteName, this.config.WhiteNameIPv6} {
			if len(listName) == 0 {
				continue
			}
			var cmd = executils.NewTimeoutCmd(30*time.Second, path, "--permanent", "--add-rich-rule=rule source ipset='"+listName+"' accept")
			cmd.WithStderr()
			err := cmd.Run()
			if err != nil {
				return fmt.Errorf("firewall-cmd add rich rule '%s': %w, output: %s", listName, err, cmd.Stderr())
			}
		}

		// reject
		for _, listName := range []string{this.config.BlackName, this.config.BlackNameIPv6} {
			if len(listName) == 0 {
				continue
			}
			var cmd = executils.NewTimeoutCmd(30*time.Second, path, "--permanent", "--add-rich-rule=rule source ipset='"+listName+"' reject")
			cmd.WithStderr()
			err := cmd.Run()
			if err != nil {
				return fmt.Errorf("firewall-cmd add rich rule '%s': %w, output: %s", listName, err, cmd.Stderr())
			}
		}

		// reload
		{
			var cmd = executils.NewTimeoutCmd(30*time.Second, path, "--reload")
			cmd.WithStderr()
			err := cmd.Run()
			if err != nil {
				return fmt.Errorf("firewall-cmd reload: %w, output: %s", err, cmd.Stderr())
			}
		}
	}

	// iptables
	if this.config.AutoAddToIPTables {
		path, err := executils.LookPath("iptables")
		if err != nil {
			return err
		}

		// accept
		for _, listName := range []string{this.config.WhiteName, this.config.WhiteNameIPv6} {
			if len(listName) == 0 {
				continue
			}

			// 检查规则是否存在
			var cmd = executils.NewTimeoutCmd(30*time.Second, path, "-C", "INPUT", "-m", "set", "--match-set", listName, "src", "-j", "ACCEPT")
			err := cmd.Run()
			var exists = err == nil

			// 添加规则
			if !exists {
				var cmd = executils.NewTimeoutCmd(30*time.Second, path, "-A", "INPUT", "-m", "set", "--match-set", listName, "src", "-j", "ACCEPT")
				cmd.WithStderr()
				err := cmd.Run()
				if err != nil {
					return fmt.Errorf("iptables add rule: %w, output: %s", err, cmd.Stderr())
				}
			}
		}

		// reject
		for _, listName := range []string{this.config.BlackName, this.config.BlackNameIPv6} {
			if len(listName) == 0 {
				continue
			}

			// 检查规则是否存在
			var cmd = executils.NewTimeoutCmd(30*time.Second, path, "-C", "INPUT", "-m", "set", "--match-set", listName, "src", "-j", "REJECT")
			err := cmd.Run()
			var exists = err == nil

			if !exists {
				var cmd = executils.NewTimeoutCmd(30*time.Second, path, "-A", "INPUT", "-m", "set", "--match-set", listName, "src", "-j", "REJECT")
				cmd.WithStderr()
				err := cmd.Run()
				if err != nil {
					return fmt.Errorf("iptables add rule: %w, output: %s", err, cmd.Stderr())
				}
			}
		}
	}

	return nil
}

func (this *IPSetAction) AddItem(listType IPListType, item *pb.IPItem) error {
	return this.runAction("addItem", listType, item)
}

func (this *IPSetAction) DeleteItem(listType IPListType, item *pb.IPItem) error {
	return this.runAction("deleteItem", listType, item)
}

func (this *IPSetAction) runAction(action string, listType IPListType, item *pb.IPItem) error {
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
		var index = strings.Index(cidr, "/")
		if index <= 0 {
			continue
		}

		// 只支持/24以下的
		if types.Int(cidr[index+1:]) < 24 {
			continue
		}

		item.IpFrom = cidr
		item.IpTo = ""
		err := this.runActionSingleIP(action, listType, item)
		if err != nil {
			return err
		}
	}
	return nil
}

func (this *IPSetAction) SetConfig(config *firewallconfigs.FirewallActionIPSetConfig) {
	this.config = config
}

func (this *IPSetAction) runActionSingleIP(action string, listType IPListType, item *pb.IPItem) error {
	if item.Type == "all" {
		return nil
	}

	var listName string
	var isIPv6 = strings.Contains(item.IpFrom, ":")

	switch listType {
	case IPListTypeWhite:
		if isIPv6 {
			listName = this.config.WhiteNameIPv6
		} else {
			listName = this.config.WhiteName
		}
	case IPListTypeBlack:
		if isIPv6 {
			listName = this.config.BlackNameIPv6
		} else {
			listName = this.config.BlackName
		}
	default:
		// 不支持的类型
		return nil
	}
	if len(listName) == 0 {
		return nil
	}

	var path = this.config.Path
	var err error
	if len(path) == 0 {
		path, err = executils.LookPath("ipset")
		if err != nil {
			// 找不到ipset命令错误只提示一次
			if this.ipsetNotfound {
				return nil
			}
			this.ipsetNotfound = true
			return err
		}
	}

	// ipset add edge_ip_list 192.168.2.32 timeout 30
	var args = []string{}
	switch action {
	case "addItem":
		args = append(args, "add")
	case "deleteItem":
		args = append(args, "del")
	}

	args = append(args, listName, item.IpFrom)
	if action == "addItem" {
		var timestamp = time.Now().Unix()
		if item.ExpiredAt > timestamp {
			args = append(args, "timeout", strconv.FormatInt(item.ExpiredAt-timestamp, 10))
		}
	}

	if runtime.GOOS == "darwin" {
		// MAC OS直接返回
		return nil
	}

	var cmd = executils.NewTimeoutCmd(30*time.Second, path, args...)
	cmd.WithStderr()
	err = cmd.Run()
	if err != nil {
		var errString = cmd.Stderr()
		if action == "deleteItem" && strings.Contains(errString, "not added") {
			return nil
		}
		return errors.New(strings.TrimSpace(errString))
	}
	return nil
}
