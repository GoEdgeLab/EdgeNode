// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.
//go:build linux
// +build linux

package firewalls

import (
	"bytes"
	"errors"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/firewalls/nftables"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/iwind/TeaGo/types"
	"net"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"
)

// check nft status, if being enabled we load it automatically
func init() {
	if teaconst.IsDaemon {
		return
	}

	if runtime.GOOS == "linux" {
		var ticker = time.NewTicker(3 * time.Minute)
		goman.New(func() {
			for range ticker.C {
				// if already ready, we break
				if nftablesIsReady {
					ticker.Stop()
					break
				}
				_, err := exec.LookPath("nft")
				if err == nil {
					nftablesFirewall, err := NewNFTablesFirewall()
					if err != nil {
						continue
					}
					currentFirewall = nftablesFirewall
					remotelogs.Println("FIREWALL", "nftables is ready")

					// fire event
					if nftablesFirewall.IsReady() {
						events.Notify(events.EventNFTablesReady)
					}

					ticker.Stop()
					break
				}
			}
		})
	}
}

var nftablesInstance *NFTablesFirewall
var nftablesIsReady = false
var nftablesFilters = []*nftablesTableDefinition{
	// we shorten the name for table name length restriction
	{Name: "edge_dft_v4", IsIPv4: true},
	{Name: "edge_dft_v6", IsIPv6: true},
}
var nftablesChainName = "input"

type nftablesTableDefinition struct {
	Name   string
	IsIPv4 bool
	IsIPv6 bool
}

func (this *nftablesTableDefinition) protocol() string {
	if this.IsIPv6 {
		return "ip6"
	}
	return "ip"
}

type blockIPItem struct {
	action         string
	ip             string
	timeoutSeconds int
}

func NewNFTablesFirewall() (*NFTablesFirewall, error) {
	var firewall = &NFTablesFirewall{
		conn:        nftables.NewConn(),
		dropIPQueue: make(chan *blockIPItem, 4096),
	}
	err := firewall.init()
	if err != nil {
		return nil, err
	}

	return firewall, nil
}

type NFTablesFirewall struct {
	conn    *nftables.Conn
	isReady bool
	version string

	allowIPv4Set *nftables.Set
	allowIPv6Set *nftables.Set

	denyIPv4Set *nftables.Set
	denyIPv6Set *nftables.Set

	firewalld *Firewalld

	dropIPQueue chan *blockIPItem
}

func (this *NFTablesFirewall) init() error {
	// check nft
	nftPath, err := exec.LookPath("nft")
	if err != nil {
		return errors.New("nft not found")
	}
	this.version = this.readVersion(nftPath)

	// table
	for _, tableDef := range nftablesFilters {
		var family nftables.TableFamily
		if tableDef.IsIPv4 {
			family = nftables.TableFamilyIPv4
		} else if tableDef.IsIPv6 {
			family = nftables.TableFamilyIPv6
		} else {
			return errors.New("invalid table family: " + types.String(tableDef))
		}
		table, err := this.conn.GetTable(tableDef.Name, family)
		if err != nil {
			if nftables.IsNotFound(err) {
				if tableDef.IsIPv4 {
					table, err = this.conn.AddIPv4Table(tableDef.Name)
				} else if tableDef.IsIPv6 {
					table, err = this.conn.AddIPv6Table(tableDef.Name)
				}
				if err != nil {
					return errors.New("create table '" + tableDef.Name + "' failed: " + err.Error())
				}
			} else {
				return errors.New("get table '" + tableDef.Name + "' failed: " + err.Error())
			}
		}
		if table == nil {
			return errors.New("can not create table '" + tableDef.Name + "'")
		}

		// chain
		var chainName = nftablesChainName
		chain, err := table.GetChain(chainName)
		if err != nil {
			if nftables.IsNotFound(err) {
				chain, err = table.AddAcceptChain(chainName)
				if err != nil {
					return errors.New("create chain '" + chainName + "' failed: " + err.Error())
				}
			} else {
				return errors.New("get chain '" + chainName + "' failed: " + err.Error())
			}
		}
		if chain == nil {
			return errors.New("can not create chain '" + chainName + "'")
		}

		// allow lo
		var loRuleName = []byte("lo")
		_, err = chain.GetRuleWithUserData(loRuleName)
		if err != nil {
			if nftables.IsNotFound(err) {
				_, err = chain.AddAcceptInterfaceRule("lo", loRuleName)
			}
			if err != nil {
				return errors.New("add 'lo' rule failed: " + err.Error())
			}
		}

		// allow set
		// "allow" should be always first
		for _, setAction := range []string{"allow", "deny"} {
			var setName = setAction + "_set"

			set, err := table.GetSet(setName)
			if err != nil {
				if nftables.IsNotFound(err) {
					var keyType nftables.SetDataType
					if tableDef.IsIPv4 {
						keyType = nftables.TypeIPAddr
					} else if tableDef.IsIPv6 {
						keyType = nftables.TypeIP6Addr
					}
					set, err = table.AddSet(setName, &nftables.SetOptions{
						KeyType:    keyType,
						HasTimeout: true,
					})
					if err != nil {
						return errors.New("create set '" + setName + "' failed: " + err.Error())
					}
				} else {
					return errors.New("get set '" + setName + "' failed: " + err.Error())
				}
			}
			if set == nil {
				return errors.New("can not create set '" + setName + "'")
			}
			if tableDef.IsIPv4 {
				if setAction == "allow" {
					this.allowIPv4Set = set
				} else {
					this.denyIPv4Set = set
				}
			} else if tableDef.IsIPv6 {
				if setAction == "allow" {
					this.allowIPv6Set = set
				} else {
					this.denyIPv6Set = set
				}
			}

			// rule
			var ruleName = []byte(setAction)
			rule, err := chain.GetRuleWithUserData(ruleName)
			if err != nil {
				if nftables.IsNotFound(err) {
					if tableDef.IsIPv4 {
						if setAction == "allow" {
							rule, err = chain.AddAcceptIPv4SetRule(setName, ruleName)
						} else {
							rule, err = chain.AddDropIPv4SetRule(setName, ruleName)
						}
					} else if tableDef.IsIPv6 {
						if setAction == "allow" {
							rule, err = chain.AddAcceptIPv6SetRule(setName, ruleName)
						} else {
							rule, err = chain.AddDropIPv6SetRule(setName, ruleName)
						}
					}
					if err != nil {
						return errors.New("add rule failed: " + err.Error())
					}
				} else {
					return errors.New("get rule failed: " + err.Error())
				}
			}
			if rule == nil {
				return errors.New("can not create rule '" + string(ruleName) + "'")
			}
		}
	}

	this.isReady = true
	nftablesIsReady = true
	nftablesInstance = this

	goman.New(func() {
		for ipItem := range this.dropIPQueue {
			switch ipItem.action {
			case "drop":
				err = this.DropSourceIP(ipItem.ip, ipItem.timeoutSeconds, false)
				if err != nil {
					remotelogs.Warn("NFTABLES", "drop ip '"+ipItem.ip+"' failed: "+err.Error())
				}
			}
		}
	})

	// load firewalld
	var firewalld = NewFirewalld()
	if firewalld.IsReady() {
		this.firewalld = firewalld
	}

	return nil
}

// Name 名称
func (this *NFTablesFirewall) Name() string {
	return "nftables"
}

// IsReady 是否已准备被调用
func (this *NFTablesFirewall) IsReady() bool {
	return this.isReady
}

// IsMock 是否为模拟
func (this *NFTablesFirewall) IsMock() bool {
	return false
}

// AllowPort 允许端口
func (this *NFTablesFirewall) AllowPort(port int, protocol string) error {
	if this.firewalld != nil {
		return this.firewalld.AllowPort(port, protocol)
	}
	return nil
}

// RemovePort 删除端口
func (this *NFTablesFirewall) RemovePort(port int, protocol string) error {
	if this.firewalld != nil {
		return this.firewalld.RemovePort(port, protocol)
	}
	return nil
}

// AllowSourceIP Allow把IP加入白名单
func (this *NFTablesFirewall) AllowSourceIP(ip string) error {
	var data = net.ParseIP(ip)
	if data == nil {
		return errors.New("invalid ip '" + ip + "'")
	}

	if strings.Contains(ip, ":") { // ipv6
		if this.allowIPv6Set == nil {
			return errors.New("ipv6 ip set is nil")
		}
		return this.allowIPv6Set.AddElement(data.To16(), nil)
	}

	// ipv4
	if this.allowIPv4Set == nil {
		return errors.New("ipv4 ip set is nil")
	}
	return this.allowIPv4Set.AddElement(data.To4(), nil)
}

// RejectSourceIP 拒绝某个源IP连接
// we did not create set for drop ip, so we reuse DropSourceIP() method here
func (this *NFTablesFirewall) RejectSourceIP(ip string, timeoutSeconds int) error {
	return this.DropSourceIP(ip, timeoutSeconds, true)
}

// DropSourceIP 丢弃某个源IP数据
func (this *NFTablesFirewall) DropSourceIP(ip string, timeoutSeconds int, async bool) error {
	var data = net.ParseIP(ip)
	if data == nil {
		return errors.New("invalid ip '" + ip + "'")
	}

	if async {
		select {
		case this.dropIPQueue <- &blockIPItem{
			action:         "drop",
			ip:             ip,
			timeoutSeconds: timeoutSeconds,
		}:
		default:
			return errors.New("drop ip queue is full")
		}
		return nil
	}

	if strings.Contains(ip, ":") { // ipv6
		if this.denyIPv6Set == nil {
			return errors.New("ipv6 ip set is nil")
		}
		return this.denyIPv6Set.AddElement(data.To16(), &nftables.ElementOptions{
			Timeout: time.Duration(timeoutSeconds) * time.Second,
		})
	}

	// ipv4
	if this.denyIPv4Set == nil {
		return errors.New("ipv4 ip set is nil")
	}
	return this.denyIPv4Set.AddElement(data.To4(), &nftables.ElementOptions{
		Timeout: time.Duration(timeoutSeconds) * time.Second,
	})
}

// RemoveSourceIP 删除某个源IP
func (this *NFTablesFirewall) RemoveSourceIP(ip string) error {
	var data = net.ParseIP(ip)
	if data == nil {
		return errors.New("invalid ip '" + ip + "'")
	}

	if strings.Contains(ip, ":") { // ipv6
		if this.denyIPv6Set != nil {
			err := this.denyIPv6Set.DeleteElement(data.To16())
			if err != nil {
				return err
			}
		}

		if this.allowIPv6Set != nil {
			err := this.allowIPv6Set.DeleteElement(data.To16())
			if err != nil {
				return err
			}
		}

		return nil
	}

	// ipv4
	if this.allowIPv4Set != nil {
		err := this.denyIPv4Set.DeleteElement(data.To4())
		if err != nil {
			return err
		}

		err = this.allowIPv4Set.DeleteElement(data.To4())
		if err != nil {
			return err
		}
	}

	return nil
}

// 读取版本号
func (this *NFTablesFirewall) readVersion(nftPath string) string {
	var cmd = exec.Command(nftPath, "--version")
	var output = &bytes.Buffer{}
	cmd.Stdout = output
	err := cmd.Run()
	if err != nil {
		return ""
	}

	var outputString = output.String()
	var versionMatches = regexp.MustCompile(`nftables v([\d.]+)`).FindStringSubmatch(outputString)
	if len(versionMatches) <= 1 {
		return ""
	}
	return versionMatches[1]
}
