package iplibrary

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/firewallconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"strconv"
	"sync"
)

var SharedActionManager = NewActionManager()

// ActionManager 动作管理器定义
type ActionManager struct {
	locker sync.Mutex

	eventMap    map[string][]ActionInterface                    // eventLevel => []instance
	configMap   map[int64]*firewallconfigs.FirewallActionConfig // id => config
	instanceMap map[int64]ActionInterface                       // id => instance
}

// NewActionManager 获取动作管理对象
func NewActionManager() *ActionManager {
	return &ActionManager{
		configMap:   map[int64]*firewallconfigs.FirewallActionConfig{},
		instanceMap: map[int64]ActionInterface{},
	}
}

// UpdateActions 更新配置
func (this *ActionManager) UpdateActions(actions []*firewallconfigs.FirewallActionConfig) {
	this.locker.Lock()
	defer this.locker.Unlock()

	// 关闭不存在的
	newActionsMap := map[int64]*firewallconfigs.FirewallActionConfig{}
	for _, action := range actions {
		newActionsMap[action.Id] = action
	}
	for _, oldAction := range this.configMap {
		_, ok := newActionsMap[oldAction.Id]
		if !ok {
			instance, ok := this.instanceMap[oldAction.Id]
			if ok {
				_ = instance.Close()
				delete(this.instanceMap, oldAction.Id)
				remotelogs.Println("IPLIBRARY/ACTION_MANAGER", "close action "+strconv.FormatInt(oldAction.Id, 10))
			}
		}
	}

	// 添加新的或者更新老的
	for _, newAction := range newActionsMap {
		oldInstance, ok := this.instanceMap[newAction.Id]
		if ok {
			// 检查配置是否一致
			oldConfigJSON, err := json.Marshal(this.configMap[newAction.Id])
			if err != nil {
				remotelogs.Error("IPLIBRARY/ACTION_MANAGER", "action "+strconv.FormatInt(newAction.Id, 10)+", type:"+newAction.Type+": "+err.Error())
				continue
			}
			newConfigJSON, err := json.Marshal(newAction)
			if err != nil {
				remotelogs.Error("IPLIBRARY/ACTION_MANAGER", "action "+strconv.FormatInt(newAction.Id, 10)+", type:"+newAction.Type+": "+err.Error())
				continue
			}
			if !bytes.Equal(newConfigJSON, oldConfigJSON) {
				_ = oldInstance.Close()

				// 重新创建
				// 之所以要重新创建，是因为前后的动作类型可能有变化，完全重建可以避免不必要的麻烦
				newInstance, err := this.createInstance(newAction)
				if err != nil {
					remotelogs.Error("IPLIBRARY/ACTION_MANAGER", "reload action "+strconv.FormatInt(newAction.Id, 10)+", type:"+newAction.Type+": "+err.Error())
					continue
				}
				remotelogs.Println("IPLIBRARY/ACTION_MANAGER", "reloaded "+strconv.FormatInt(newAction.Id, 10)+", type:"+newAction.Type)
				this.instanceMap[newAction.Id] = newInstance
			}
		} else {
			// 创建
			instance, err := this.createInstance(newAction)
			if err != nil {
				remotelogs.Error("IPLIBRARY/ACTION_MANAGER", "load new action "+strconv.FormatInt(newAction.Id, 10)+", type:"+newAction.Type+": "+err.Error())
				continue
			}
			remotelogs.Println("IPLIBRARY/ACTION_MANAGER", "loaded action "+strconv.FormatInt(newAction.Id, 10)+", type:"+newAction.Type)
			this.instanceMap[newAction.Id] = instance
		}
	}

	// 更新配置
	this.configMap = newActionsMap
	this.eventMap = map[string][]ActionInterface{}
	for _, action := range this.configMap {
		instance, ok := this.instanceMap[action.Id]
		if !ok {
			continue
		}

		var instances = this.eventMap[action.EventLevel]
		instances = append(instances, instance)
		this.eventMap[action.EventLevel] = instances
	}
}

// FindEventActions 查找事件对应的动作
func (this *ActionManager) FindEventActions(eventLevel string) []ActionInterface {
	this.locker.Lock()
	defer this.locker.Unlock()
	return this.eventMap[eventLevel]
}

// AddItem 执行添加IP动作
func (this *ActionManager) AddItem(listType IPListType, item *pb.IPItem) {
	instances, ok := this.eventMap[item.EventLevel]
	if ok {
		for _, instance := range instances {
			err := instance.AddItem(listType, item)
			if err != nil {
				remotelogs.Error("IPLIBRARY/ACTION_MANAGER", "add item '"+fmt.Sprintf("%d", item.Id)+"': "+err.Error())
			}
		}
	}
}

// DeleteItem 执行删除IP动作
func (this *ActionManager) DeleteItem(listType IPListType, item *pb.IPItem) {
	instances, ok := this.eventMap[item.EventLevel]
	if ok {
		for _, instance := range instances {
			err := instance.DeleteItem(listType, item)
			if err != nil {
				remotelogs.Error("IPLIBRARY/ACTION_MANAGER", "delete item '"+fmt.Sprintf("%d", item.Id)+"': "+err.Error())
			}
		}
	}
}

func (this *ActionManager) createInstance(config *firewallconfigs.FirewallActionConfig) (ActionInterface, error) {
	var instance ActionInterface
	switch config.Type {
	case firewallconfigs.FirewallActionTypeIPSet:
		instance = NewIPSetAction()
	case firewallconfigs.FirewallActionTypeFirewalld:
		instance = NewFirewalldAction()
	case firewallconfigs.FirewallActionTypeIPTables:
		instance = NewIPTablesAction()
	case firewallconfigs.FirewallActionTypeScript:
		instance = NewScriptAction()
	case firewallconfigs.FirewallActionTypeHTTPAPI:
		instance = NewHTTPAPIAction()
	case firewallconfigs.FirewallActionTypeHTML:
		instance = NewHTMLAction()
	}
	if instance == nil {
		return nil, errors.New("can not create instance for type '" + config.Type + "'")
	}
	err := instance.Init(config)
	if err != nil {
		// 如果是警告错误，我们只是提示
		if !IsFatalError(err) {
			remotelogs.Error("IPLIBRARY/ACTION_MANAGER/CREATE_INSTANCE", "init '"+config.Type+"' failed: "+err.Error())
		} else {
			return nil, err
		}
	}
	return instance, nil
}
