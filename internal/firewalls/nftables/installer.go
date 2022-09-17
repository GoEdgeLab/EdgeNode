// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package nftables

import (
	"errors"
	"github.com/TeaOSLab/EdgeCommon/pkg/nodeconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	executils "github.com/TeaOSLab/EdgeNode/internal/utils/exec"
	"github.com/iwind/TeaGo/logs"
	"os"
	"os/exec"
	"runtime"
	"time"
)

func init() {
	events.On(events.EventReload, func() {
		// linux only
		if runtime.GOOS != "linux" {
			return
		}

		nodeConfig, err := nodeconfigs.SharedNodeConfig()
		if err != nil {
			return
		}

		if nodeConfig == nil || !nodeConfig.AutoInstallNftables {
			return
		}

		if os.Getgid() == 0 { // root user only
			_, err := exec.LookPath("nft")
			if err == nil {
				return
			}
			goman.New(func() {
				err := NewInstaller().Install()
				if err != nil {
					// 不需要传到API节点
					logs.Println("[NFTABLES]install nftables failed: " + err.Error())
				}
			})
		}
	})
}

type Installer struct {
}

func NewInstaller() *Installer {
	return &Installer{}
}

func (this *Installer) Install() error {
	// linux only
	if runtime.GOOS != "linux" {
		return nil
	}

	// 检查是否已经存在
	_, err := exec.LookPath("nft")
	if err == nil {
		return nil
	}

	var cmd *executils.Cmd

	// check dnf
	dnfExe, err := exec.LookPath("dnf")
	if err == nil {
		cmd = executils.NewCmd(dnfExe, "-y", "install", "nftables")
	}

	// check apt
	if cmd == nil {
		aptExe, err := exec.LookPath("apt")
		if err == nil {
			cmd = executils.NewCmd(aptExe, "install", "nftables")
		}
	}

	// check yum
	if cmd == nil {
		yumExe, err := exec.LookPath("yum")
		if err == nil {
			cmd = executils.NewCmd(yumExe, "-y", "install", "nftables")
		}
	}

	if cmd == nil {
		return nil
	}

	cmd.WithTimeout(10 * time.Minute)
	cmd.WithStderr()
	err = cmd.Run()
	if err != nil {
		return errors.New(err.Error() + ": " + cmd.Stderr())
	}

	remotelogs.Println("NFTABLES", "installed nftables with command '"+cmd.String()+"' successfully")

	return nil
}
