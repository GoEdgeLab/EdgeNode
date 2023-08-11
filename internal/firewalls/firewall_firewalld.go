// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package firewalls

import (
	"fmt"
	"github.com/TeaOSLab/EdgeNode/internal/conns"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	executils "github.com/TeaOSLab/EdgeNode/internal/utils/exec"
	"github.com/iwind/TeaGo/types"
	"strings"
	"time"
)

type firewalldCmd struct {
	cmd    *executils.Cmd
	denyIP string
}

type Firewalld struct {
	BaseFirewall

	isReady  bool
	exe      string
	cmdQueue chan *firewalldCmd
}

func NewFirewalld() *Firewalld {
	var firewalld = &Firewalld{
		cmdQueue: make(chan *firewalldCmd, 4096),
	}

	path, err := executils.LookPath("firewall-cmd")
	if err == nil && len(path) > 0 {
		var cmd = executils.NewTimeoutCmd(3*time.Second, path, "--state")
		err := cmd.Run()
		if err == nil {
			firewalld.exe = path
			// TODO check firewalld status with 'firewall-cmd --state' (running or not running),
			//      but we should recover the state when firewalld state changes, maybe check it every minutes

			firewalld.isReady = true
			firewalld.init()
		}
	}

	return firewalld
}

func (this *Firewalld) init() {
	goman.New(func() {
		for c := range this.cmdQueue {
			var cmd = c.cmd
			err := cmd.Run()
			if err != nil {
				if strings.HasPrefix(err.Error(), "Warning:") {
					continue
				}
				remotelogs.Warn("FIREWALL", "run command failed '"+cmd.String()+"': "+err.Error())
			} else {
				// 关闭连接
				if len(c.denyIP) > 0 {
					conns.SharedMap.CloseIPConns(c.denyIP)
				}
			}
		}
	})
}

// Name 名称
func (this *Firewalld) Name() string {
	return "firewalld"
}

func (this *Firewalld) IsReady() bool {
	return this.isReady
}

// IsMock 是否为模拟
func (this *Firewalld) IsMock() bool {
	return false
}

func (this *Firewalld) AllowPort(port int, protocol string) error {
	if !this.isReady {
		return nil
	}
	var cmd = executils.NewTimeoutCmd(10*time.Second, this.exe, "--add-port="+types.String(port)+"/"+protocol)
	this.pushCmd(cmd, "")
	return nil
}

func (this *Firewalld) AllowPortRangesPermanently(portRanges [][2]int, protocol string) error {
	for _, portRange := range portRanges {
		var port = this.PortRangeString(portRange, protocol)

		{
			var cmd = executils.NewTimeoutCmd(10*time.Second, this.exe, "--add-port="+port, "--permanent")
			this.pushCmd(cmd, "")
		}

		{
			var cmd = executils.NewTimeoutCmd(10*time.Second, this.exe, "--add-port="+port)
			this.pushCmd(cmd, "")
		}
	}

	return nil
}

func (this *Firewalld) RemovePort(port int, protocol string) error {
	if !this.isReady {
		return nil
	}
	var cmd = executils.NewTimeoutCmd(10*time.Second, this.exe, "--remove-port="+types.String(port)+"/"+protocol)
	this.pushCmd(cmd, "")
	return nil
}

func (this *Firewalld) RemovePortRangePermanently(portRange [2]int, protocol string) error {
	var port = this.PortRangeString(portRange, protocol)

	{
		var cmd = executils.NewTimeoutCmd(10*time.Second, this.exe, "--remove-port="+port, "--permanent")
		this.pushCmd(cmd, "")
	}

	{
		var cmd = executils.NewTimeoutCmd(10*time.Second, this.exe, "--remove-port="+port)
		this.pushCmd(cmd, "")
	}

	return nil
}

func (this *Firewalld) PortRangeString(portRange [2]int, protocol string) string {
	if portRange[0] == portRange[1] {
		return types.String(portRange[0]) + "/" + protocol
	} else {
		return types.String(portRange[0]) + "-" + types.String(portRange[1]) + "/" + protocol
	}
}

func (this *Firewalld) RejectSourceIP(ip string, timeoutSeconds int) error {
	if !this.isReady {
		return nil
	}

	// 避免短时间内重复添加
	if this.checkLatestIP(ip) {
		return nil
	}

	var family = "ipv4"
	if strings.Contains(ip, ":") {
		family = "ipv6"
	}
	var args = []string{"--add-rich-rule=rule family='" + family + "' source address='" + ip + "' reject"}
	if timeoutSeconds > 0 {
		args = append(args, "--timeout="+types.String(timeoutSeconds)+"s")
	}
	var cmd = executils.NewTimeoutCmd(10*time.Second, this.exe, args...)
	this.pushCmd(cmd, ip)
	return nil
}

func (this *Firewalld) DropSourceIP(ip string, timeoutSeconds int, async bool) error {
	if !this.isReady {
		return nil
	}

	// 避免短时间内重复添加
	if async && this.checkLatestIP(ip) {
		return nil
	}

	var family = "ipv4"
	if strings.Contains(ip, ":") {
		family = "ipv6"
	}
	var args = []string{"--add-rich-rule=rule family='" + family + "' source address='" + ip + "' drop"}
	if timeoutSeconds > 0 {
		args = append(args, "--timeout="+types.String(timeoutSeconds)+"s")
	}
	var cmd = executils.NewTimeoutCmd(10*time.Second, this.exe, args...)
	if async {
		this.pushCmd(cmd, ip)
		return nil
	}

	// 关闭连接
	defer conns.SharedMap.CloseIPConns(ip)

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("run command failed '%s': %w", cmd.String(), err)
	}
	return nil
}

func (this *Firewalld) RemoveSourceIP(ip string) error {
	if !this.isReady {
		return nil
	}

	var family = "ipv4"
	if strings.Contains(ip, ":") {
		family = "ipv6"
	}
	for _, action := range []string{"reject", "drop"} {
		var args = []string{"--remove-rich-rule=rule family='" + family + "' source address='" + ip + "' " + action}
		var cmd = executils.NewTimeoutCmd(10*time.Second, this.exe, args...)
		this.pushCmd(cmd, "")
	}
	return nil
}

func (this *Firewalld) pushCmd(cmd *executils.Cmd, denyIP string) {
	select {
	case this.cmdQueue <- &firewalldCmd{cmd: cmd, denyIP: denyIP}:
	default:
		// we discard the command
	}
}
