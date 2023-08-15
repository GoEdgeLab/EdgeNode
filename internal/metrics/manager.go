// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package metrics

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"strconv"
	"sync"
)

var SharedManager = NewManager()

func init() {
	if !teaconst.IsMain {
		return
	}

	events.OnClose(func() {
		SharedManager.Quit()
	})
}

type Manager struct {
	isQuiting bool

	taskMap         map[int64]*Task    // itemId => *Task
	categoryTaskMap map[string][]*Task // category => []*Task
	locker          sync.RWMutex

	hasHTTPMetrics bool
	hasTCPMetrics  bool
	hasUDPMetrics  bool
}

func NewManager() *Manager {
	return &Manager{
		taskMap:         map[int64]*Task{},
		categoryTaskMap: map[string][]*Task{},
	}
}

func (this *Manager) Update(items []*serverconfigs.MetricItemConfig) {
	if this.isQuiting {
		return
	}

	this.locker.Lock()
	defer this.locker.Unlock()

	var newMap = map[int64]*serverconfigs.MetricItemConfig{}
	for _, item := range items {
		newMap[item.Id] = item
	}

	// 停用以前的 或 修改现在的
	for itemId, task := range this.taskMap {
		newItem, ok := newMap[itemId]
		if !ok || !newItem.IsOn { // 停用以前的
			remotelogs.Println("METRIC_MANAGER", "stop task '"+strconv.FormatInt(itemId, 10)+"'")
			err := task.Stop()
			if err != nil {
				remotelogs.Error("METRIC_MANAGER", "stop task '"+strconv.FormatInt(itemId, 10)+"' failed: "+err.Error())
			}
			delete(this.taskMap, itemId)
		} else { // 更新已存在的
			if newItem.Version != task.item.Version {
				remotelogs.Println("METRIC_MANAGER", "update task '"+strconv.FormatInt(itemId, 10)+"'")
				task.item = newItem
			}
		}
	}

	// 启动新的
	for _, newItem := range items {
		if !newItem.IsOn {
			continue
		}
		_, ok := this.taskMap[newItem.Id]
		if !ok {
			remotelogs.Println("METRIC_MANAGER", "start task '"+strconv.FormatInt(newItem.Id, 10)+"'")
			task := NewTask(newItem)
			err := task.Init()
			if err != nil {
				remotelogs.Error("METRIC_MANAGER", "initialized task failed: "+err.Error())
				continue
			}
			err = task.Start()
			if err != nil {
				remotelogs.Error("METRIC_MANAGER", "start task failed: "+err.Error())
				continue
			}
			this.taskMap[newItem.Id] = task
		}
	}

	// 按分类存放
	this.hasHTTPMetrics = false
	this.hasTCPMetrics = false
	this.hasUDPMetrics = false
	this.categoryTaskMap = map[string][]*Task{}
	for _, task := range this.taskMap {
		var tasks = this.categoryTaskMap[task.item.Category]
		tasks = append(tasks, task)
		this.categoryTaskMap[task.item.Category] = tasks

		switch task.item.Category {
		case serverconfigs.MetricItemCategoryHTTP:
			this.hasHTTPMetrics = true
		case serverconfigs.MetricItemCategoryTCP:
			this.hasTCPMetrics = true
		case serverconfigs.MetricItemCategoryUDP:
			this.hasUDPMetrics = true
		}
	}
}

// Add 添加数据
func (this *Manager) Add(obj MetricInterface) {
	if this.isQuiting {
		return
	}

	this.locker.RLock()
	var tasks = this.categoryTaskMap[obj.MetricCategory()]
	this.locker.RUnlock()

	for _, task := range tasks {
		task.Add(obj)
	}
}

func (this *Manager) HasHTTPMetrics() bool {
	return this.hasHTTPMetrics
}

func (this *Manager) HasTCPMetrics() bool {
	return this.hasTCPMetrics
}

func (this *Manager) HasUDPMetrics() bool {
	return this.hasUDPMetrics
}

// Quit 退出管理器
func (this *Manager) Quit() {
	this.isQuiting = true

	remotelogs.Println("METRIC_MANAGER", "quit")

	this.locker.Lock()
	for _, task := range this.taskMap {
		_ = task.Stop()
	}
	this.taskMap = map[int64]*Task{}
	this.locker.Unlock()
}
