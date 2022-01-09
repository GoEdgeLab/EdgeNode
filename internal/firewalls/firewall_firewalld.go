// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package firewalls

import (
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/iwind/TeaGo/types"
	"os/exec"
	"strings"
)

type Firewalld struct {
	isReady  bool
	exe      string
	cmdQueue chan *exec.Cmd
}

func NewFirewalld() *Firewalld {
	var firewalld = &Firewalld{
		cmdQueue: make(chan *exec.Cmd, 2048),
	}

	path, err := exec.LookPath("firewall-cmd")
	if err == nil && len(path) > 0 {
		var cmd = exec.Command(path, "-V")
		err := cmd.Run()
		if err == nil {
			firewalld.exe = path
			firewalld.isReady = true
			firewalld.init()
		}
	}

	return firewalld
}

func (this *Firewalld) init() {
	goman.New(func() {
		for cmd := range this.cmdQueue {
			err := cmd.Run()
			if err != nil {
				if strings.HasPrefix(err.Error(), "Warning:") {
					continue
				}
				remotelogs.Warn("FIREWALL", "run command failed '"+cmd.String()+"': "+err.Error())
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

func (this *Firewalld) AllowPort(port int, protocol string) error {
	if !this.isReady {
		return nil
	}
	var cmd = exec.Command(this.exe, "--add-port="+types.String(port)+"/"+protocol)
	this.pushCmd(cmd)
	return nil
}

func (this *Firewalld) RemovePort(port int, protocol string) error {
	if !this.isReady {
		return nil
	}
	var cmd = exec.Command(this.exe, "--remove-port="+types.String(port)+"/"+protocol)
	this.pushCmd(cmd)
	return nil
}

func (this *Firewalld) RejectSourceIP(ip string, timeoutSeconds int) error {
	if !this.isReady {
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
	var cmd = exec.Command(this.exe, args...)
	this.pushCmd(cmd)
	return nil
}

func (this *Firewalld) DropSourceIP(ip string, timeoutSeconds int) error {
	if !this.isReady {
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
	var cmd = exec.Command(this.exe, args...)
	this.pushCmd(cmd)
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
		var cmd = exec.Command(this.exe, args...)
		this.pushCmd(cmd)
	}
	return nil
}

func (this *Firewalld) pushCmd(cmd *exec.Cmd) {
	select {
	case this.cmdQueue <- cmd:
	default:
		// we discard the command
	}
}
