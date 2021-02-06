package iplibrary

import (
	"bytes"
	"errors"
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/firewallconfigs"
	"os/exec"
	"runtime"
	"strconv"
	"time"
)

// IPSet动作
// 相关命令：
//   - 利用Firewalld管理set：
//       - 添加：firewall-cmd --permanent --new-ipset=edge_ip_list --type=hash:ip --option="timeout=0"
//       - 删除：firewall-cmd --permanent --delete-ipset=edge_ip_list
//       - 重载：firewall-cmd --reload
//       - firewalld+ipset: firewall-cmd --permanent --add-rich-rule="rule source ipset='edge_ip_list' reject"
//   - 利用IPTables管理set：
//       - 添加：iptables -A INPUT -m set --match-set edge_ip_list src -j REJECT
//   - 添加Item：ipset add edge_ip_list 192.168.2.32 timeout 30
//   - 删除Item: ipset del edge_ip_list 192.168.2.32
//   - 创建set：ipset create edge_ip_list hash:ip timeout 0
type IPSetAction struct {
	BaseAction

	config *firewallconfigs.FirewallActionIPSetConfig
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
		path, err := exec.LookPath("ipset")
		if err != nil {
			return err
		}
		{
			cmd := exec.Command(path, "create", this.config.WhiteName, "hash:ip", "timeout", "0")
			stderr := bytes.NewBuffer([]byte{})
			cmd.Stderr = stderr
			err := cmd.Run()
			if err != nil {
				output := stderr.Bytes()
				if !bytes.Contains(output, []byte("already exists")) {
					return errors.New("create ipset '" + this.config.WhiteName + "': " + err.Error() + ", output: " + string(output))
				} else {
					err = nil
				}
			}
		}
		{
			cmd := exec.Command(path, "create", this.config.BlackName, "hash:ip", "timeout", "0")
			stderr := bytes.NewBuffer([]byte{})
			cmd.Stderr = stderr
			err := cmd.Run()
			if err != nil {
				output := stderr.Bytes()
				if !bytes.Contains(output, []byte("already exists")) {
					return errors.New("create ipset '" + this.config.BlackName + "': " + err.Error() + ", output: " + string(output))
				} else {
					err = nil
				}
			}
		}
	}

	// firewalld
	if this.config.AutoAddToFirewalld {
		path, err := exec.LookPath("firewall-cmd")
		if err != nil {
			return err
		}

		{
			cmd := exec.Command(path, "--permanent", "--new-ipset="+this.config.WhiteName, "--type=hash:ip", "--option=timeout=0", "--option=maxelem=1000000")
			stderr := bytes.NewBuffer([]byte{})
			cmd.Stderr = stderr
			err := cmd.Run()
			if err != nil {
				output := stderr.Bytes()
				if bytes.Contains(output, []byte("NAME_CONFLICT")) {
					err = nil
				} else {
					return errors.New("firewall-cmd add ipset '" + this.config.WhiteName + "': " + err.Error() + ", output: " + string(output))
				}
			}
		}

		{
			cmd := exec.Command(path, "--permanent", "--add-rich-rule=rule source ipset='"+this.config.WhiteName+"' accept")
			stderr := bytes.NewBuffer([]byte{})
			cmd.Stderr = stderr
			err := cmd.Run()
			if err != nil {
				output := stderr.Bytes()
				return errors.New("firewall-cmd add rich rule '" + this.config.WhiteName + "': " + err.Error() + ", output: " + string(output))
			}
		}

		{
			cmd := exec.Command(path, "--permanent", "--new-ipset="+this.config.BlackName, "--type=hash:ip", "--option=timeout=0", "--option=maxelem=1000000")
			stderr := bytes.NewBuffer([]byte{})
			cmd.Stderr = stderr
			err := cmd.Run()
			if err != nil {
				output := stderr.Bytes()
				if bytes.Contains(output, []byte("NAME_CONFLICT")) {
					err = nil
				} else {
					return errors.New("firewall-cmd add ipset '" + this.config.BlackName + "': " + err.Error() + ", output: " + string(output))
				}
			}
		}

		{
			cmd := exec.Command(path, "--permanent", "--add-rich-rule=rule source ipset='"+this.config.BlackName+"' reject")
			stderr := bytes.NewBuffer([]byte{})
			cmd.Stderr = stderr
			err := cmd.Run()
			if err != nil {
				output := stderr.Bytes()
				return errors.New("firewall-cmd add rich rule '" + this.config.WhiteName + "': " + err.Error() + ", output: " + string(output))
			}
		}

		// reload
		{
			cmd := exec.Command(path, "--reload")
			stderr := bytes.NewBuffer([]byte{})
			cmd.Stderr = stderr
			err := cmd.Run()
			if err != nil {
				output := stderr.Bytes()
				return errors.New("firewall-cmd reload: " + err.Error() + ", output: " + string(output))
			}
		}
	}

	// iptables
	if this.config.AutoAddToIPTables {
		path, err := exec.LookPath("iptables")
		if err != nil {
			return err
		}

		{
			cmd := exec.Command(path, "-A", "INPUT", "-m", "set", "--match-set", this.config.WhiteName, "src", "-j", "ACCEPT")
			stderr := bytes.NewBuffer([]byte{})
			cmd.Stderr = stderr
			err := cmd.Run()
			if err != nil {
				output := stderr.Bytes()
				return errors.New("iptables add rule: " + err.Error() + ", output: " + string(output))
			}
		}

		{
			cmd := exec.Command(path, "-A", "INPUT", "-m", "set", "--match-set", this.config.BlackName, "src", "-j", "REJECT")
			stderr := bytes.NewBuffer([]byte{})
			cmd.Stderr = stderr
			err := cmd.Run()
			if err != nil {
				output := stderr.Bytes()
				return errors.New("iptables add rule: " + err.Error() + ", output: " + string(output))
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
		item.IpFrom = cidr
		item.IpTo = ""
		err := this.runActionSingleIP(action, listType, item)
		if err != nil {
			return err
		}
	}
	return nil
}

func (this *IPSetAction) runActionSingleIP(action string, listType IPListType, item *pb.IPItem) error {
	if item.Type == "all" {
		return nil
	}

	listName := ""
	switch listType {
	case IPListTypeWhite:
		listName = this.config.WhiteName
	case IPListTypeBlack:
		listName = this.config.BlackName
	default:
		// 不支持的类型
		return nil
	}
	if len(listName) == 0 {
		return errors.New("empty list name for '" + listType + "'")
	}

	path := this.config.Path
	var err error
	if len(path) == 0 {
		path, err = exec.LookPath("ipset")
		if err != nil {
			return err
		}
	}

	// ipset add edge_ip_list 192.168.2.32 timeout 30
	args := []string{}
	switch action {
	case "addItem":
		args = append(args, "add")
	case "deleteItem":
		args = append(args, "del")
	}
	args = append(args, listName, item.IpFrom)
	timestamp := time.Now().Unix()
	if item.ExpiredAt > timestamp {
		args = append(args, "timeout", strconv.FormatInt(item.ExpiredAt-timestamp, 10))
	}

	//logs.Println(args)

	if runtime.GOOS == "darwin" {
		// MAC OS直接返回
		return nil
	}

	cmd := exec.Command(path, args...)
	return cmd.Run()
}
