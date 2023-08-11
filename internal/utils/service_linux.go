//go:build linux
// +build linux

package utils

import (
	"errors"
	"fmt"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	executils "github.com/TeaOSLab/EdgeNode/internal/utils/exec"
	"github.com/iwind/TeaGo/Tea"
	"github.com/iwind/TeaGo/files"
	"os"
	"os/exec"
	"regexp"
	"time"
)

var systemdServiceFile = "/etc/systemd/system/edge-node.service"
var initServiceFile = "/etc/init.d/" + teaconst.SystemdServiceName

// Install 安装服务
func (this *ServiceManager) Install(exePath string, args []string) error {
	if os.Getgid() != 0 {
		return errors.New("only root users can install the service")
	}

	systemd, err := executils.LookPath("systemctl")
	if err != nil {
		return this.installInitService(exePath, args)
	}

	return this.installSystemdService(systemd, exePath, args)
}

// Start 启动服务
func (this *ServiceManager) Start() error {
	if os.Getgid() != 0 {
		return errors.New("only root users can start the service")
	}

	if files.NewFile(systemdServiceFile).Exists() {
		systemd, err := executils.LookPath("systemctl")
		if err != nil {
			return err
		}

		return exec.Command(systemd, "start", teaconst.SystemdServiceName+".service").Start()
	}
	return exec.Command("service", teaconst.ProcessName, "start").Start()
}

// Uninstall 删除服务
func (this *ServiceManager) Uninstall() error {
	if os.Getgid() != 0 {
		return errors.New("only root users can uninstall the service")
	}

	if files.NewFile(systemdServiceFile).Exists() {
		systemd, err := executils.LookPath("systemctl")
		if err != nil {
			return err
		}

		// disable service
		_ = executils.NewTimeoutCmd(10*time.Second, systemd, "disable", teaconst.SystemdServiceName+".service").Start()

		// reload
		_ = executils.NewTimeoutCmd(10*time.Second, systemd, "daemon-reload").Start()

		return files.NewFile(systemdServiceFile).Delete()
	}

	f := files.NewFile(initServiceFile)
	if f.Exists() {
		return f.Delete()
	}
	return nil
}

// install init service
func (this *ServiceManager) installInitService(exePath string, args []string) error {
	shortName := teaconst.SystemdServiceName
	scriptFile := Tea.Root + "/scripts/" + shortName
	if !files.NewFile(scriptFile).Exists() {
		return errors.New("'scripts/" + shortName + "' file not exists")
	}

	data, err := os.ReadFile(scriptFile)
	if err != nil {
		return err
	}

	data = regexp.MustCompile("INSTALL_DIR=.+").ReplaceAll(data, []byte("INSTALL_DIR="+Tea.Root))
	err = os.WriteFile(initServiceFile, data, 0777)
	if err != nil {
		return err
	}

	chkCmd, err := executils.LookPath("chkconfig")
	if err != nil {
		return err
	}

	err = exec.Command(chkCmd, "--add", teaconst.ProcessName).Start()
	if err != nil {
		return err
	}

	return nil
}

// install systemd service
func (this *ServiceManager) installSystemdService(systemd, exePath string, args []string) error {
	var shortName = teaconst.SystemdServiceName
	var longName = "GoEdge Node" // TODO 将来可以修改

	var desc = `### BEGIN INIT INFO
# Provides:          ` + shortName + `
# Required-Start:    $all
# Required-Stop:
# Default-Start:     2 3 4 5
# Default-Stop:
# Short-Description: ` + longName + ` Service
### END INIT INFO

[Unit]
Description=` + longName + ` Service
Before=shutdown.target
After=network-online.target

[Service]
Type=simple
Restart=always
RestartSec=1s
ExecStart=` + exePath + ` daemon
ExecStop=` + exePath + ` stop
ExecReload=` + exePath + ` reload

[Install]
WantedBy=multi-user.target`

	// write file
	err := os.WriteFile(systemdServiceFile, []byte(desc), 0777)
	if err != nil {
		return err
	}

	// stop current systemd service if running
	_ = executils.NewTimeoutCmd(10*time.Second, systemd, "stop", shortName+".service").Start()

	// reload
	_ = executils.NewTimeoutCmd(10*time.Second, systemd, "daemon-reload").Start()

	// enable
	var cmd = executils.NewTimeoutCmd(10*time.Second, systemd, "enable", shortName+".service")
	cmd.WithStderr()
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("%w: %s", err, cmd.Stderr())
	}
	return nil
}
