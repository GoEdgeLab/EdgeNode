package serverconfigs

import (
	"github.com/TeaOSLab/EdgeNode/internal/configs/serverconfigs/scheduling"
	"github.com/TeaOSLab/EdgeNode/internal/configs/serverconfigs/shared"
	"sync"
)

// 反向代理设置
type ReverseProxyConfig struct {
	IsOn       bool                  `yaml:"isOn" json:"isOn"`             // 是否启用 TODO
	Origins    []*OriginServerConfig `yaml:"origins" json:"origins"`       // 源站列表
	Scheduling *SchedulingConfig     `yaml:"scheduling" json:"scheduling"` // 调度算法选项

	hasOrigins         bool
	schedulingIsBackup bool
	schedulingObject   scheduling.SchedulingInterface
	schedulingLocker   sync.Mutex
}

// 初始化
func (this *ReverseProxyConfig) Init() error {
	this.hasOrigins = len(this.Origins) > 0

	for _, origin := range this.Origins {
		err := origin.Init()
		if err != nil {
			return err
		}
	}

	// scheduling
	this.SetupScheduling(false)

	return nil
}

// 取得下一个可用的后端服务
func (this *ReverseProxyConfig) NextOrigin(call *shared.RequestCall) *OriginServerConfig {
	this.schedulingLocker.Lock()
	defer this.schedulingLocker.Unlock()

	if this.schedulingObject == nil {
		return nil
	}

	if this.Scheduling != nil && call != nil && call.Options != nil {
		for k, v := range this.Scheduling.Options {
			call.Options[k] = v
		}
	}

	candidate := this.schedulingObject.Next(call)
	if candidate == nil {
		// 启用备用服务器
		if !this.schedulingIsBackup {
			this.SetupScheduling(true)

			candidate = this.schedulingObject.Next(call)
			if candidate == nil {
				return nil
			}
		}

		if candidate == nil {
			return nil
		}
	}

	return candidate.(*OriginServerConfig)
}

// 设置调度算法
func (this *ReverseProxyConfig) SetupScheduling(isBackup bool) {
	if !isBackup {
		this.schedulingLocker.Lock()
		defer this.schedulingLocker.Unlock()
	}
	this.schedulingIsBackup = isBackup

	if this.Scheduling == nil {
		this.schedulingObject = &scheduling.RandomScheduling{}
	} else {
		typeCode := this.Scheduling.Code
		s := scheduling.FindSchedulingType(typeCode)
		if s == nil {
			this.Scheduling = nil
			this.schedulingObject = &scheduling.RandomScheduling{}
		} else {
			this.schedulingObject = s["instance"].(scheduling.SchedulingInterface)
		}
	}

	for _, origin := range this.Origins {
		if origin.IsOn && !origin.IsDown {
			if isBackup && origin.IsBackup {
				this.schedulingObject.Add(origin)
			} else if !isBackup && !origin.IsBackup {
				this.schedulingObject.Add(origin)
			}
		}
	}

	this.schedulingObject.Start()
}
