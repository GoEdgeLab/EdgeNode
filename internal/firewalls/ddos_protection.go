// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.
//go:build linux

package firewalls

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/TeaOSLab/EdgeCommon/pkg/nodeconfigs"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/ddosconfigs"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/firewalls/nftables"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	executils "github.com/TeaOSLab/EdgeNode/internal/utils/exec"
	"github.com/TeaOSLab/EdgeNode/internal/zero"
	"github.com/iwind/TeaGo/lists"
	"github.com/iwind/TeaGo/types"
	stringutil "github.com/iwind/TeaGo/utils/string"
	"net"
	"strings"
	"sync"
	"time"
)

var SharedDDoSProtectionManager = NewDDoSProtectionManager()

func init() {
	if !teaconst.IsMain {
		return
	}

	events.On(events.EventReload, func() {
		if nftablesInstance == nil {
			return
		}

		nodeConfig, _ := nodeconfigs.SharedNodeConfig()
		if nodeConfig != nil {
			err := SharedDDoSProtectionManager.Apply(nodeConfig.DDoSProtection)
			if err != nil {
				remotelogs.Error("FIREWALL", "apply DDoS protection failed: "+err.Error())
			}
		}
	})

	events.On(events.EventNFTablesReady, func() {
		nodeConfig, _ := nodeconfigs.SharedNodeConfig()
		if nodeConfig != nil {
			err := SharedDDoSProtectionManager.Apply(nodeConfig.DDoSProtection)
			if err != nil {
				remotelogs.Error("FIREWALL", "apply DDoS protection failed: "+err.Error())
			}
		}
	})
}

// DDoSProtectionManager DDoS防护
type DDoSProtectionManager struct {
	lastAllowIPList []string
	lastConfig      []byte

	locker sync.Mutex
}

// NewDDoSProtectionManager 获取新对象
func NewDDoSProtectionManager() *DDoSProtectionManager {
	return &DDoSProtectionManager{}
}

// Apply 应用配置
func (this *DDoSProtectionManager) Apply(config *ddosconfigs.ProtectionConfig) error {
	// 加锁防止并发更改
	if !this.locker.TryLock() {
		return nil
	}
	defer this.locker.Unlock()

	// 同集群节点IP白名单
	var allowIPListChanged = false
	nodeConfig, _ := nodeconfigs.SharedNodeConfig()
	if nodeConfig != nil {
		var allowIPList = nodeConfig.AllowedIPs
		if !utils.EqualStrings(allowIPList, this.lastAllowIPList) {
			allowIPListChanged = true
			this.lastAllowIPList = allowIPList
		}
	}

	// 对比配置
	configJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("encode config to json failed: %w", err)
	}
	if !allowIPListChanged && bytes.Equal(this.lastConfig, configJSON) {
		return nil
	}
	remotelogs.Println("FIREWALL", "change DDoS protection config")

	if len(nftables.NftExePath()) == 0 {
		return errors.New("can not find nft command")
	}

	if nftablesInstance == nil {
		if config == nil || !config.IsOn() {
			return nil
		}
		return errors.New("nftables instance should not be nil")
	}

	if config == nil {
		// TCP
		err := this.removeTCPRules()
		if err != nil {
			return err
		}

		// TODO other protocols

		return nil
	}

	// TCP
	if config.TCP == nil {
		err := this.removeTCPRules()
		if err != nil {
			return err
		}
	} else {
		// allow ip list
		var allowIPList = []string{}
		for _, ipConfig := range config.TCP.AllowIPList {
			allowIPList = append(allowIPList, ipConfig.IP)
		}
		for _, ip := range this.lastAllowIPList {
			if !lists.ContainsString(allowIPList, ip) {
				allowIPList = append(allowIPList, ip)
			}
		}
		err = this.updateAllowIPList(allowIPList)
		if err != nil {
			return err
		}

		// tcp
		if config.TCP.IsOn {
			err := this.addTCPRules(config.TCP)
			if err != nil {
				return err
			}
		} else {
			err := this.removeTCPRules()
			if err != nil {
				return err
			}
		}
	}

	this.lastConfig = configJSON

	return nil
}

// 添加TCP规则
func (this *DDoSProtectionManager) addTCPRules(tcpConfig *ddosconfigs.TCPConfig) error {
	var nftExe = nftables.NftExePath()
	if len(nftExe) == 0 {
		return nil
	}

	// 检查nft版本不能小于0.9
	if len(nftablesInstance.version) > 0 && stringutil.VersionCompare("0.9", nftablesInstance.version) > 0 {
		return nil
	}

	var ports = []int32{}
	for _, portConfig := range tcpConfig.Ports {
		if !lists.ContainsInt32(ports, portConfig.Port) {
			ports = append(ports, portConfig.Port)
		}
	}
	if len(ports) == 0 {
		ports = []int32{80, 443}
	}

	for _, filter := range nftablesFilters {
		chain, oldRules, err := this.getRules(filter)
		if err != nil {
			return fmt.Errorf("get old rules failed: %w", err)
		}

		var protocol = filter.protocol()

		// max connections
		var maxConnections = tcpConfig.MaxConnections
		if maxConnections <= 0 {
			maxConnections = nodeconfigs.DefaultTCPMaxConnections
			if maxConnections <= 0 {
				maxConnections = 100000
			}
		}

		// max connections per ip
		var maxConnectionsPerIP = tcpConfig.MaxConnectionsPerIP
		if maxConnectionsPerIP <= 0 {
			maxConnectionsPerIP = nodeconfigs.DefaultTCPMaxConnectionsPerIP
			if maxConnectionsPerIP <= 0 {
				maxConnectionsPerIP = 100000
			}
		}

		// new connections rate (minutely)
		var newConnectionsMinutelyRate = tcpConfig.NewConnectionsMinutelyRate
		if newConnectionsMinutelyRate <= 0 {
			newConnectionsMinutelyRate = nodeconfigs.DefaultTCPNewConnectionsMinutelyRate
			if newConnectionsMinutelyRate <= 0 {
				newConnectionsMinutelyRate = 100000
			}
		}
		var newConnectionsMinutelyRateBlockTimeout = tcpConfig.NewConnectionsMinutelyRateBlockTimeout
		if newConnectionsMinutelyRateBlockTimeout < 0 {
			newConnectionsMinutelyRateBlockTimeout = 0
		}

		// new connections rate (secondly)
		var newConnectionsSecondlyRate = tcpConfig.NewConnectionsSecondlyRate
		if newConnectionsSecondlyRate <= 0 {
			newConnectionsSecondlyRate = nodeconfigs.DefaultTCPNewConnectionsSecondlyRate
			if newConnectionsSecondlyRate <= 0 {
				newConnectionsSecondlyRate = 10000
			}
		}
		var newConnectionsSecondlyRateBlockTimeout = tcpConfig.NewConnectionsSecondlyRateBlockTimeout
		if newConnectionsSecondlyRateBlockTimeout < 0 {
			newConnectionsSecondlyRateBlockTimeout = 0
		}

		// 检查是否有变化
		var hasChanges = false
		for _, port := range ports {
			if !this.existsRule(oldRules, []string{"tcp", types.String(port), "maxConnections", types.String(maxConnections)}) {
				hasChanges = true
				break
			}
			if !this.existsRule(oldRules, []string{"tcp", types.String(port), "maxConnectionsPerIP", types.String(maxConnectionsPerIP)}) {
				hasChanges = true
				break
			}
			if !this.existsRule(oldRules, []string{"tcp", types.String(port), "newConnectionsRate", types.String(newConnectionsMinutelyRate), types.String(newConnectionsMinutelyRateBlockTimeout)}) {
				hasChanges = true
				break
			}
			if !this.existsRule(oldRules, []string{"tcp", types.String(port), "newConnectionsSecondlyRate", types.String(newConnectionsSecondlyRate), types.String(newConnectionsSecondlyRateBlockTimeout)}) {
				hasChanges = true
				break
			}
		}

		if !hasChanges {
			// 检查是否有多余的端口
			var oldPorts = this.getTCPPorts(oldRules)
			if !this.eqPorts(ports, oldPorts) {
				hasChanges = true
			}
		}

		if !hasChanges {
			return nil
		}

		// 先清空所有相关规则
		err = this.removeOldTCPRules(chain, oldRules)
		if err != nil {
			return fmt.Errorf("delete old rules failed: %w", err)
		}

		// 添加新规则
		for _, port := range ports {
			if maxConnections > 0 {
				var cmd = executils.NewTimeoutCmd(10*time.Second, nftExe, "add", "rule", protocol, filter.Name, nftablesChainName, "tcp", "dport", types.String(port), "ct", "count", "over", types.String(maxConnections), "counter", "drop", "comment", this.encodeUserData([]string{"tcp", types.String(port), "maxConnections", types.String(maxConnections)}))
				cmd.WithStderr()
				err = cmd.Run()
				if err != nil {
					return fmt.Errorf("add nftables rule '%s' failed: %w (%s)", cmd.String(), err, cmd.Stderr())
				}
			}

			// TODO 让用户选择是drop还是reject
			if maxConnectionsPerIP > 0 {
				var cmd = executils.NewTimeoutCmd(10*time.Second, nftExe, "add", "rule", protocol, filter.Name, nftablesChainName, "tcp", "dport", types.String(port), "meter", "meter-"+protocol+"-"+types.String(port)+"-max-connections", "{ "+protocol+" saddr ct count over "+types.String(maxConnectionsPerIP)+" }", "counter", "drop", "comment", this.encodeUserData([]string{"tcp", types.String(port), "maxConnectionsPerIP", types.String(maxConnectionsPerIP)}))
				cmd.WithStderr()
				err := cmd.Run()
				if err != nil {
					return fmt.Errorf("add nftables rule '%s' failed: %w (%s)", cmd.String(), err, cmd.Stderr())
				}
			}

			// 超过一定速率就drop或者加入黑名单（分钟）
			// TODO 让用户选择是drop还是reject
			if newConnectionsMinutelyRate > 0 {
				if newConnectionsMinutelyRateBlockTimeout > 0 {
					var cmd = executils.NewTimeoutCmd(10*time.Second, nftExe, "add", "rule", protocol, filter.Name, nftablesChainName, "tcp", "dport", types.String(port), "ct", "state", "new", "meter", "meter-"+protocol+"-"+types.String(port)+"-new-connections-rate", "{ "+protocol+" saddr limit rate over "+types.String(newConnectionsMinutelyRate)+"/minute burst "+types.String(newConnectionsMinutelyRate+3)+" packets }", "add", "@deny_set", "{"+protocol+" saddr timeout "+types.String(newConnectionsMinutelyRateBlockTimeout)+"s}", "comment", this.encodeUserData([]string{"tcp", types.String(port), "newConnectionsRate", types.String(newConnectionsMinutelyRate), types.String(newConnectionsMinutelyRateBlockTimeout)}))
					cmd.WithStderr()
					err := cmd.Run()
					if err != nil {
						return fmt.Errorf("add nftables rule '%s' failed: %w (%s)", cmd.String(), err, cmd.Stderr())
					}
				} else {
					var cmd = executils.NewTimeoutCmd(10*time.Second, nftExe, "add", "rule", protocol, filter.Name, nftablesChainName, "tcp", "dport", types.String(port), "ct", "state", "new", "meter", "meter-"+protocol+"-"+types.String(port)+"-new-connections-rate", "{ "+protocol+" saddr limit rate over "+types.String(newConnectionsMinutelyRate)+"/minute burst "+types.String(newConnectionsMinutelyRate+3)+" packets }" /**"add", "@deny_set", "{"+protocol+" saddr}",**/, "counter", "drop", "comment", this.encodeUserData([]string{"tcp", types.String(port), "newConnectionsRate", "0"}))
					cmd.WithStderr()
					err := cmd.Run()
					if err != nil {
						return fmt.Errorf("add nftables rule '%s' failed: %w (%s)", cmd.String(), err, cmd.Stderr())
					}
				}
			}

			// 超过一定速率就drop或者加入黑名单（秒）
			// TODO 让用户选择是drop还是reject
			if newConnectionsSecondlyRate > 0 {
				if newConnectionsSecondlyRateBlockTimeout > 0 {
					var cmd = executils.NewTimeoutCmd(10*time.Second, nftExe, "add", "rule", protocol, filter.Name, nftablesChainName, "tcp", "dport", types.String(port), "ct", "state", "new", "meter", "meter-"+protocol+"-"+types.String(port)+"-new-connections-secondly-rate", "{ "+protocol+" saddr limit rate over "+types.String(newConnectionsSecondlyRate)+"/second burst "+types.String(newConnectionsSecondlyRate+3)+" packets }", "add", "@deny_set", "{"+protocol+" saddr timeout "+types.String(newConnectionsSecondlyRateBlockTimeout)+"s}", "comment", this.encodeUserData([]string{"tcp", types.String(port), "newConnectionsSecondlyRate", types.String(newConnectionsSecondlyRate), types.String(newConnectionsSecondlyRateBlockTimeout)}))
					cmd.WithStderr()
					err := cmd.Run()
					if err != nil {
						return fmt.Errorf("add nftables rule '%s' failed: %w (%s)", cmd.String(), err, cmd.Stderr())
					}
				} else {
					var cmd = executils.NewTimeoutCmd(10*time.Second, nftExe, "add", "rule", protocol, filter.Name, nftablesChainName, "tcp", "dport", types.String(port), "ct", "state", "new", "meter", "meter-"+protocol+"-"+types.String(port)+"-new-connections-secondly-rate", "{ "+protocol+" saddr limit rate over "+types.String(newConnectionsSecondlyRate)+"/second burst "+types.String(newConnectionsSecondlyRate+3)+" packets }" /**"add", "@deny_set", "{"+protocol+" saddr}",**/, "counter", "drop", "comment", this.encodeUserData([]string{"tcp", types.String(port), "newConnectionsSecondlyRate", "0"}))
					cmd.WithStderr()
					err := cmd.Run()
					if err != nil {
						return fmt.Errorf("add nftables rule '%s' failed: %w (%s)", cmd.String(), err, cmd.Stderr())
					}
				}
			}
		}
	}

	return nil
}

// 删除TCP规则
func (this *DDoSProtectionManager) removeTCPRules() error {
	for _, filter := range nftablesFilters {
		chain, rules, err := this.getRules(filter)

		// TCP
		err = this.removeOldTCPRules(chain, rules)
		if err != nil {
			return err
		}
	}

	return nil
}

// 组合user data
// 数据中不能包含字母、数字、下划线以外的数据
func (this *DDoSProtectionManager) encodeUserData(attrs []string) string {
	if attrs == nil {
		return ""
	}

	return "ZZ" + strings.Join(attrs, "_") + "ZZ"
}

// 解码user data
func (this *DDoSProtectionManager) decodeUserData(data []byte) []string {
	if len(data) == 0 {
		return nil
	}

	var dataCopy = make([]byte, len(data))
	copy(dataCopy, data)

	var separatorLen = 2
	var index1 = bytes.Index(dataCopy, []byte{'Z', 'Z'})
	if index1 < 0 {
		return nil
	}

	dataCopy = dataCopy[index1+separatorLen:]
	var index2 = bytes.LastIndex(dataCopy, []byte{'Z', 'Z'})
	if index2 < 0 {
		return nil
	}

	var s = string(dataCopy[:index2])
	var pieces = strings.Split(s, "_")
	for index, piece := range pieces {
		pieces[index] = strings.TrimSpace(piece)
	}
	return pieces
}

// 清除规则
func (this *DDoSProtectionManager) removeOldTCPRules(chain *nftables.Chain, rules []*nftables.Rule) error {
	for _, rule := range rules {
		var pieces = this.decodeUserData(rule.UserData())
		if len(pieces) < 4 {
			continue
		}
		if pieces[0] != "tcp" {
			continue
		}
		switch pieces[2] {
		case "maxConnections", "maxConnectionsPerIP", "newConnectionsRate", "newConnectionsSecondlyRate":
			err := chain.DeleteRule(rule)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// 根据参数检查规则是否存在
func (this *DDoSProtectionManager) existsRule(rules []*nftables.Rule, attrs []string) (exists bool) {
	if len(attrs) == 0 {
		return false
	}
	for _, oldRule := range rules {
		var pieces = this.decodeUserData(oldRule.UserData())
		if len(attrs) != len(pieces) {
			continue
		}
		var isSame = true
		for index, piece := range pieces {
			if strings.TrimSpace(piece) != attrs[index] {
				isSame = false
				break
			}
		}
		if isSame {
			return true
		}
	}
	return false
}

// 获取规则中的端口号
func (this *DDoSProtectionManager) getTCPPorts(rules []*nftables.Rule) []int32 {
	var ports = []int32{}
	for _, rule := range rules {
		var pieces = this.decodeUserData(rule.UserData())
		if len(pieces) != 4 {
			continue
		}
		if pieces[0] != "tcp" {
			continue
		}
		var port = types.Int32(pieces[1])
		if port > 0 && !lists.ContainsInt32(ports, port) {
			ports = append(ports, port)
		}
	}
	return ports
}

// 检查端口是否一样
func (this *DDoSProtectionManager) eqPorts(ports1 []int32, ports2 []int32) bool {
	if len(ports1) != len(ports2) {
		return false
	}

	var portMap = map[int32]bool{}
	for _, port := range ports2 {
		portMap[port] = true
	}

	for _, port := range ports1 {
		_, ok := portMap[port]
		if !ok {
			return false
		}
	}
	return true
}

// 查找Table
func (this *DDoSProtectionManager) getTable(filter *nftablesTableDefinition) (*nftables.Table, error) {
	var family nftables.TableFamily
	if filter.IsIPv4 {
		family = nftables.TableFamilyIPv4
	} else if filter.IsIPv6 {
		family = nftables.TableFamilyIPv6
	} else {
		return nil, errors.New("table '" + filter.Name + "' should be IPv4 or IPv6")
	}
	return nftablesInstance.conn.GetTable(filter.Name, family)
}

// 查找所有规则
func (this *DDoSProtectionManager) getRules(filter *nftablesTableDefinition) (*nftables.Chain, []*nftables.Rule, error) {
	table, err := this.getTable(filter)
	if err != nil {
		return nil, nil, fmt.Errorf("get table failed: %w", err)
	}
	chain, err := table.GetChain(nftablesChainName)
	if err != nil {
		return nil, nil, fmt.Errorf("get chain failed: %w", err)
	}
	rules, err := chain.GetRules()
	return chain, rules, err
}

// 更新白名单
func (this *DDoSProtectionManager) updateAllowIPList(allIPList []string) error {
	if nftablesInstance == nil {
		return nil
	}

	var allMap = map[string]zero.Zero{}
	for _, ip := range allIPList {
		allMap[ip] = zero.New()
	}

	for _, set := range []*nftables.Set{nftablesInstance.allowIPv4Set, nftablesInstance.allowIPv6Set} {
		var isIPv4 = set == nftablesInstance.allowIPv4Set
		var isIPv6 = !isIPv4

		// 现有的
		oldList, err := set.GetIPElements()
		if err != nil {
			return err
		}
		var oldMap = map[string]zero.Zero{} // ip=> zero
		for _, ip := range oldList {
			oldMap[ip] = zero.New()

			if (utils.IsIPv4(ip) && isIPv4) || (utils.IsIPv6(ip) && isIPv6) {
				_, ok := allMap[ip]
				if !ok {
					// 不存在则删除
					err = set.DeleteIPElement(ip)
					if err != nil {
						return fmt.Errorf("delete ip element '%s' failed: %w", ip, err)
					}
				}
			}
		}

		// 新增的
		for _, ip := range allIPList {
			var ipObj = net.ParseIP(ip)
			if ipObj == nil {
				continue
			}
			if (utils.IsIPv4(ip) && isIPv4) || (utils.IsIPv6(ip) && isIPv6) {
				_, ok := oldMap[ip]
				if !ok {
					// 不存在则添加
					err = set.AddIPElement(ip, nil, false)
					if err != nil {
						return fmt.Errorf("add ip '%s' failed: %w", ip, err)
					}
				}
			}
		}
	}

	return nil
}
