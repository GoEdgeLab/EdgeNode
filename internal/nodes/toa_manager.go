package nodes

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/nodeconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/iwind/TeaGo/Tea"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"
)

var sharedTOAManager = NewTOAManager()

func init() {
	events.On(events.EventReload, func() {
		err := sharedTOAManager.Run(sharedNodeConfig.TOA)
		if err != nil {
			remotelogs.Error("TOA", err.Error())
		}
	})
}

type TOAManager struct {
	config *nodeconfigs.TOAConfig
	pid    int
	conn   net.Conn
}

func NewTOAManager() *TOAManager {
	return &TOAManager{}
}

func (this *TOAManager) Run(config *nodeconfigs.TOAConfig) error {
	this.config = config

	if this.pid > 0 {
		remotelogs.Println("TOA", "stopping ...")
		err := this.Quit()
		if err != nil {
			remotelogs.Error("TOA", "quit error: "+err.Error())
		}
		_ = this.conn.Close()
		this.conn = nil
		this.pid = 0
	}

	if !config.IsOn {
		return nil
	}

	binPath := Tea.Root + "/edge-toa/edge-toa" // TODO 可以做成配置
	_, err := os.Stat(binPath)
	if err != nil {
		return err
	}
	remotelogs.Println("TOA", "starting ...")
	remotelogs.Println("TOA", "args: "+strings.Join(config.AsArgs(), " "))
	cmd := exec.Command(binPath, config.AsArgs()...)
	err = cmd.Start()
	if err != nil {
		return err
	}
	this.pid = cmd.Process.Pid

	go func() { _ = cmd.Wait() }()

	return nil
}

func (this *TOAManager) Config() *nodeconfigs.TOAConfig {
	return this.config
}

func (this *TOAManager) Quit() error {
	return this.SendMsg("quit:0")
}

func (this *TOAManager) SendMsg(msg string) error {
	if this.config == nil {
		return nil
	}

	if this.conn != nil {
		_, err := this.conn.Write([]byte(msg + "\n"))
		if err != nil {
			this.conn = nil
		}
		return err
	}

	conn, err := net.DialTimeout("unix", this.config.SockFile(), 1*time.Second)
	if err != nil {
		return err
	}
	this.conn = conn
	_, err = this.conn.Write([]byte(msg + "\n"))
	return err
}
