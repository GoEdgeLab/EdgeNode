// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package nodes

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/trackers"
	"github.com/iwind/TeaGo/Tea"
	"sync"
	"time"
)

var SharedOriginStateManager = NewOriginStateManager()

func init() {
	if !teaconst.IsMain {
		return
	}

	events.On(events.EventLoaded, func() {
		goman.New(func() {
			SharedOriginStateManager.Start()
		})
	})
	events.On(events.EventQuit, func() {
		SharedOriginStateManager.Stop()
	})
}

const (
	maxOriginStates = 512 // 最多可以监控的源站状态数量
)

// OriginStateManager 源站状态管理
type OriginStateManager struct {
	stateMap map[int64]*OriginState // originId => *OriginState

	ticker *time.Ticker
	locker sync.RWMutex
}

// NewOriginStateManager 获取新管理对象
func NewOriginStateManager() *OriginStateManager {
	return &OriginStateManager{
		stateMap: map[int64]*OriginState{},
		ticker:   time.NewTicker(60 * time.Second),
	}
}

// Start 启动
func (this *OriginStateManager) Start() {
	if Tea.IsTesting() {
		this.ticker = time.NewTicker(10 * time.Second)
	}
	for range this.ticker.C {
		err := this.Loop()
		if err != nil {
			remotelogs.Error("ORIGIN_MANAGER", err.Error())
		}
	}
}

func (this *OriginStateManager) Stop() {
	if this.ticker != nil {
		this.ticker.Stop()
	}
}

// Loop 单次循环检查
func (this *OriginStateManager) Loop() error {
	var nodeConfig = sharedNodeConfig // 复制
	if nodeConfig == nil {
		return nil
	}

	var tr = trackers.Begin("CHECK_ORIGIN_STATES")
	defer tr.End()

	var currentStates = []*OriginState{}
	this.locker.Lock()
	for originId, state := range this.stateMap {
		// 检查Origin是否正在使用
		var originConfig = nodeConfig.FindOrigin(originId)
		if originConfig == nil || !originConfig.IsOn {
			delete(this.stateMap, originId)
			continue
		}
		state.Config = originConfig
		currentStates = append(currentStates, state)
	}
	this.locker.Unlock()

	if len(currentStates) == 0 {
		return nil
	}

	var count = len(currentStates)
	var wg = &sync.WaitGroup{}
	wg.Add(count)
	for _, state := range currentStates {
		go func(state *OriginState) {
			defer wg.Done()
			conn, _, err := OriginConnect(state.Config, 0, "", state.TLSHost)
			if err == nil {
				_ = conn.Close()

				// 已经恢复正常
				this.locker.Lock()
				state.Config.IsOk = true
				delete(this.stateMap, state.Config.Id)
				this.locker.Unlock()

				var reverseProxy = state.ReverseProxy
				if reverseProxy != nil {
					reverseProxy.ResetScheduling()
				}
			}
		}(state)
	}
	wg.Wait()

	return nil
}

// Fail 添加失败的源站
func (this *OriginStateManager) Fail(origin *serverconfigs.OriginConfig, tlsHost string, reverseProxy *serverconfigs.ReverseProxyConfig, callback func()) {
	if origin == nil || origin.Id <= 0 {
		return
	}

	this.locker.Lock()
	state, ok := this.stateMap[origin.Id]
	var timestamp = time.Now().Unix()
	if ok {
		if state.UpdatedAt < timestamp-300 { // N 分钟之后重新计数
			state.CountFails = 0
			state.Config.IsOk = true
		}

		state.TLSHost = tlsHost
		state.CountFails++
		state.Config = origin
		state.ReverseProxy = reverseProxy
		state.UpdatedAt = timestamp

		if origin.IsOk {
			origin.IsOk = state.CountFails < 5 // 超过 N 次之后认为是异常

			if !origin.IsOk {
				if callback != nil {
					callback()
				}
			}
		}
	} else {
		// 同时最多监控 N 个源站地址
		if len(this.stateMap) < maxOriginStates {
			this.stateMap[origin.Id] = &OriginState{
				CountFails:   1,
				Config:       origin,
				TLSHost:      tlsHost,
				ReverseProxy: reverseProxy,
				UpdatedAt:    timestamp,
			}
		}

		origin.IsOk = true
	}
	this.locker.Unlock()
}

// Success 添加成功的源站
func (this *OriginStateManager) Success(origin *serverconfigs.OriginConfig, callback func()) {
	if origin == nil || origin.Id <= 0 {
		return
	}

	if !origin.IsOk {
		if callback != nil {
			defer callback()
		}
	}

	origin.IsOk = true
	this.locker.Lock()
	delete(this.stateMap, origin.Id)
	this.locker.Unlock()
}

// IsAvailable 检查是否正常
func (this *OriginStateManager) IsAvailable(originId int64) bool {
	if originId <= 0 {
		return true
	}

	this.locker.RLock()
	_, ok := this.stateMap[originId]
	this.locker.RUnlock()

	return !ok
}
