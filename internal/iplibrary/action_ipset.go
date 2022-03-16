package iplibrary

import (
	"bytes"
	"errors"
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/firewallconfigs"
	"github.com/iwind/TeaGo/types"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// IPSetAction IPSet动作
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
//   - 查看统计：ipset -t list edge_black_list
//   - 删除set：ipset destroy edge_black_list
type IPSetAction struct {
	BaseAction

	config   *firewallconfigs.FirewallActionIPSetConfig
	errorBuf *bytes.Buffer

	ipsetNotfound bool
}

func NewIPSetAction() *IPSetAction {
	return &IPSetAction{
		errorBuf: &bytes.Buffer{},
	}
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

		// ipv4
		for _, listName := range []string{this.config.WhiteName, this.config.BlackName} {
			if len(listName) == 0 {
				continue
			}
			var cmd = exec.Command(path, "create", listName, "hash:ip", "timeout", "0", "maxelem", "1000000")
			var stderr = bytes.NewBuffer([]byte{})
			cmd.Stderr = stderr
			err := cmd.Run()
			if err != nil {
				var output = stderr.Bytes()
				if !bytes.Contains(output, []byte("already exists")) {
					return errors.New("create ipset '" + listName + "': " + err.Error() + ", output: " + string(output))
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
			var cmd = exec.Command(path, "create", listName, "hash:ip", "family", "inet6", "timeout", "0", "maxelem", "1000000")
			var stderr = bytes.NewBuffer([]byte{})
			cmd.Stderr = stderr
			err := cmd.Run()
			if err != nil {
				var output = stderr.Bytes()
				if !bytes.Contains(output, []byte("already exists")) {
					return errors.New("create ipset '" + listName + "': " + err.Error() + ", output: " + string(output))
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

		// ipv4
		for _, listName := range []string{this.config.WhiteName, this.config.BlackName} {
			if len(listName) == 0 {
				continue
			}
			cmd := exec.Command(path, "--permanent", "--new-ipset="+listName, "--type=hash:ip", "--option=timeout=0", "--option=maxelem=1000000")
			stderr := bytes.NewBuffer([]byte{})
			cmd.Stderr = stderr
			err := cmd.Run()
			if err != nil {
				output := stderr.Bytes()
				if bytes.Contains(output, []byte("NAME_CONFLICT")) {
					err = nil
				} else {
					return errors.New("firewall-cmd add ipset '" + listName + "': " + err.Error() + ", output: " + string(output))
				}
			}
		}

		// ipv6
		for _, listName := range []string{this.config.WhiteNameIPv6, this.config.BlackNameIPv6} {
			if len(listName) == 0 {
				continue
			}
			cmd := exec.Command(path, "--permanent", "--new-ipset="+listName, "--type=hash:ip", "--option=family=inet6", "--option=timeout=0", "--option=maxelem=1000000")
			stderr := bytes.NewBuffer([]byte{})
			cmd.Stderr = stderr
			err := cmd.Run()
			if err != nil {
				var output = stderr.Bytes()
				if bytes.Contains(output, []byte("NAME_CONFLICT")) {
					err = nil
				} else {
					return errors.New("firewall-cmd add ipset '" + listName + "': " + err.Error() + ", output: " + string(output))
				}
			}
		}

		// accept
		for _, listName := range []string{this.config.WhiteName, this.config.WhiteNameIPv6} {
			if len(listName) == 0 {
				continue
			}
			var cmd = exec.Command(path, "--permanent", "--add-rich-rule=rule source ipset='"+listName+"' accept")
			var stderr = bytes.NewBuffer([]byte{})
			cmd.Stderr = stderr
			err := cmd.Run()
			if err != nil {
				var output = stderr.Bytes()
				return errors.New("firewall-cmd add rich rule '" + listName + "': " + err.Error() + ", output: " + string(output))
			}
		}

		// reject
		for _, listName := range []string{this.config.BlackName, this.config.BlackNameIPv6} {
			if len(listName) == 0 {
				continue
			}
			var cmd = exec.Command(path, "--permanent", "--add-rich-rule=rule source ipset='"+listName+"' reject")
			var stderr = bytes.NewBuffer([]byte{})
			cmd.Stderr = stderr
			err := cmd.Run()
			if err != nil {
				var output = stderr.Bytes()
				return errors.New("firewall-cmd add rich rule '" + listName + "': " + err.Error() + ", output: " + string(output))
			}
		}

		// reload
		{
			cmd := exec.Command(path, "--reload")
			stderr := bytes.NewBuffer([]byte{})
			cmd.Stderr = stderr
			err := cmd.Run()
			if err != nil {
				var output = stderr.Bytes()
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

		// accept
		for _, listName := range []string{this.config.WhiteName, this.config.WhiteNameIPv6} {
			if len(listName) == 0 {
				continue
			}

			// 检查规则是否存在
			var cmd = exec.Command(path, "-C", "INPUT", "-m", "set", "--match-set", listName, "src", "-j", "ACCEPT")
			err := cmd.Run()
			var exists = err == nil

			// 添加规则
			if !exists {
				var cmd = exec.Command(path, "-A", "INPUT", "-m", "set", "--match-set", listName, "src", "-j", "ACCEPT")
				var stderr = bytes.NewBuffer([]byte{})
				cmd.Stderr = stderr
				err := cmd.Run()
				if err != nil {
					var output = stderr.Bytes()
					return errors.New("iptables add rule: " + err.Error() + ", output: " + string(output))
				}
			}
		}

		// reject
		for _, listName := range []string{this.config.BlackName, this.config.BlackNameIPv6} {
			if len(listName) == 0 {
				continue
			}

			// 检查规则是否存在
			var cmd = exec.Command(path, "-C", "INPUT", "-m", "set", "--match-set", listName, "src", "-j", "REJECT")
			err := cmd.Run()
			var exists = err == nil

			if !exists {
				var cmd = exec.Command(path, "-A", "INPUT", "-m", "set", "--match-set", listName, "src", "-j", "REJECT")
				var stderr = bytes.NewBuffer([]byte{})
				cmd.Stderr = stderr
				err := cmd.Run()
				if err != nil {
					var output = stderr.Bytes()
					return errors.New("iptables add rule: " + err.Error() + ", output: " + string(output))
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

	var listName = ""
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
		path, err = exec.LookPath("ipset")
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

	this.errorBuf.Reset()
	var cmd = exec.Command(path, args...)
	cmd.Stderr = this.errorBuf
	err = cmd.Run()
	if err != nil {
		var errString = this.errorBuf.String()
		if action == "deleteItem" && strings.Contains(errString, "not added") {
			return nil
		}
		return errors.New(strings.TrimSpace(errString))
	}
	return nil
}
