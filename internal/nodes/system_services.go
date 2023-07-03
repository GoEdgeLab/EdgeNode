package nodes

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/TeaOSLab/EdgeCommon/pkg/nodeconfigs"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	executils "github.com/TeaOSLab/EdgeNode/internal/utils/exec"
	"github.com/iwind/TeaGo/maps"
	"os"
	"runtime"
	"time"
)

func init() {
	if !teaconst.IsMain {
		return
	}

	var manager = NewSystemServiceManager()
	events.On(events.EventReload, func() {
		goman.New(func() {
			err := manager.Setup()
			if err != nil {
				remotelogs.Error("SYSTEM_SERVICE", "setup system services failed: "+err.Error())
			}
		})
	})
}

// SystemServiceManager 系统服务管理
type SystemServiceManager struct {
	lastIsOn int // -1, 0, 1
}

func NewSystemServiceManager() *SystemServiceManager {
	return &SystemServiceManager{}
}

func (this *SystemServiceManager) Setup() error {
	if sharedNodeConfig == nil || !sharedNodeConfig.IsOn {
		return nil
	}

	if len(sharedNodeConfig.SystemServices) == 0 {
		return nil
	}

	systemdParams, ok := sharedNodeConfig.SystemServices[nodeconfigs.SystemServiceTypeSystemd]
	if ok {
		err := this.setupSystemd(systemdParams)
		if err != nil {
			return err
		}
	}

	return nil
}

func (this *SystemServiceManager) setupSystemd(params maps.Map) error {
	// 只有在Linux下运行
	if runtime.GOOS != "linux" {
		return nil
	}

	if params == nil {
		params = maps.Map{}
	}
	data, err := json.Marshal(params)
	if err != nil {
		return err
	}

	var config = &nodeconfigs.SystemdServiceConfig{}
	err = json.Unmarshal(data, config)
	if err != nil {
		return err
	}

	// 检查当前的service
	systemctl, err := executils.LookPath("systemctl")
	if err != nil {
		return err
	}
	if len(systemctl) == 0 {
		return errors.New("can not find 'systemctl' on the system")
	}

	// 记录上次状态
	var isOnInt int
	if config.IsOn {
		isOnInt = 1
	} else {
		isOnInt = 0
	}

	if this.lastIsOn == isOnInt {
		return nil
	}
	defer func() {
		this.lastIsOn = isOnInt
	}()

	var shortName = teaconst.SystemdServiceName
	var cmd = executils.NewTimeoutCmd(10*time.Second, systemctl, "is-enabled", shortName)
	cmd.WithStdout()
	err = cmd.Run()
	var hasInstalled = err == nil
	if config.IsOn {
		exe, err := os.Executable()
		if err != nil {
			return err
		}

		// 检查文件路径是否变化
		if hasInstalled && cmd.Stdout() == "enabled" {
			data, err := os.ReadFile("/etc/systemd/system/" + teaconst.SystemdServiceName + ".service")
			if err == nil && bytes.Index(data, []byte(exe)) > 0 {
				return nil
			}
		}

		// 安装服务
		var manager = utils.NewServiceManager(shortName, teaconst.ProductName)
		err = manager.Install(exe, []string{})
		if err != nil {
			return err
		}

		// 启动服务
		goman.New(func() {
			time.Sleep(5 * time.Second)
			_ = executils.NewTimeoutCmd(30*time.Second, systemctl, "start", teaconst.SystemdServiceName).Start()
		})
	} else {
		if hasInstalled {
			var manager = utils.NewServiceManager(shortName, teaconst.ProductName)
			err = manager.Uninstall()
			if err != nil {
				return err
			}
		}
	}

	return nil
}
