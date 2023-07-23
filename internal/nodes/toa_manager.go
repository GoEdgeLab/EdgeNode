package nodes

import (
	"errors"
	"github.com/TeaOSLab/EdgeCommon/pkg/nodeconfigs"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	executils "github.com/TeaOSLab/EdgeNode/internal/utils/exec"
	"github.com/iwind/TeaGo/Tea"
	"net"
	"os"
	"strings"
	"time"
)

var sharedTOAManager = NewTOAManager()

func init() {
	if !teaconst.IsMain {
		return
	}

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
		if this.conn != nil {
			_ = this.conn.Close()
		}
		this.conn = nil
		this.pid = 0
	}

	if !config.IsOn {
		return nil
	}

	var binPath = Tea.Root + "/edge-toa/edge-toa" // TODO 可以做成配置
	_, err := os.Stat(binPath)
	if err != nil {
		return err
	}
	remotelogs.Println("TOA", "starting ...")
	remotelogs.Println("TOA", "args: "+strings.Join(config.AsArgs(), " "))
	var cmd = executils.NewCmd(binPath, config.AsArgs()...)
	err = cmd.Start()
	if err != nil {
		return err
	}
	var process = cmd.Process()
	if process == nil {
		return errors.New("start failed")
	}
	this.pid = process.Pid

	goman.New(func() {
		_ = cmd.Wait()
	})

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
			_ = this.conn.Close()
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
