package nodes

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/TeaOSLab/EdgeCommon/pkg/nodeconfigs"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/iwind/TeaGo/maps"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"time"
)

func init() {
	var manager = NewSystemServiceManager()
	events.On(events.EventReload, func() {
		err := manager.Setup()
		if err != nil {
			remotelogs.Error("SYSTEM_SERVICE", "setup system services failed: "+err.Error())
		}
	})
}

// 系统服务管理
type SystemServiceManager struct {
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
	config := &nodeconfigs.SystemdServiceConfig{}
	err = json.Unmarshal(data, config)
	if err != nil {
		return err
	}

	// 检查当前的service
	systemctl, err := exec.LookPath("systemctl")
	if err != nil {
		return err
	}
	if len(systemctl) == 0 {
		return errors.New("can not find 'systemctl' on the system")
	}
	cmd := utils.NewCommandExecutor()
	shortName := teaconst.SystemdServiceName
	cmd.Add(systemctl, "is-enabled", shortName)
	output, err := cmd.Run()
	if err != nil {
		return err
	}
	if config.IsOn {
		exe, err := os.Executable()
		if err != nil {
			return err
		}

		// 启动Service
		go func() {
			time.Sleep(5 * time.Second)
			_ = exec.Command(systemctl, "start", teaconst.SystemdServiceName).Start()
		}()

		if output == "enabled" {
			// 检查文件路径是否变化
			data, err := ioutil.ReadFile("/etc/systemd/system/" + teaconst.SystemdServiceName + ".service")
			if err == nil && bytes.Index(data, []byte(exe)) > 0 {
				return nil
			}
		}
		manager := utils.NewServiceManager(shortName, teaconst.ProductName)
		err = manager.Install(exe, []string{})
		if err != nil {
			return err
		}
	} else {
		manager := utils.NewServiceManager(shortName, teaconst.ProductName)
		err = manager.Uninstall()
		if err != nil {
			return err
		}
	}

	return nil
}
