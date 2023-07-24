// Copyright 2023 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package nodes

import (
	"encoding/json"
	"errors"
	"github.com/TeaOSLab/EdgeCommon/pkg/nodeconfigs"
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/ddosconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/firewalls"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/TeaOSLab/EdgeNode/internal/iplibrary"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/rpc"
	"github.com/TeaOSLab/EdgeNode/internal/trackers"
	"github.com/iwind/TeaGo/Tea"
	"github.com/iwind/TeaGo/types"
	"os"
	"time"
)

// 循环
func (this *Node) loopTasks() error {
	var tr = trackers.Begin("CHECK_NODE_CONFIG_CHANGES")
	defer tr.End()

	// 检查api.yaml是否存在
	var apiConfigFile = Tea.ConfigFile("api.yaml")
	_, err := os.Stat(apiConfigFile)
	if err != nil {
		return nil
	}

	rpcClient, err := rpc.SharedRPC()
	if err != nil {
		return errors.New("create rpc client failed: " + err.Error())
	}

	tasksResp, err := rpcClient.NodeTaskRPC.FindNodeTasks(rpcClient.Context(), &pb.FindNodeTasksRequest{
		Version: this.lastTaskVersion,
	})
	if err != nil {
		if rpc.IsConnError(err) && !Tea.IsTesting() {
			return nil
		}
		return errors.New("read node tasks failed: " + err.Error())
	}
	for _, task := range tasksResp.NodeTasks {
		err := this.execTask(rpcClient, task)
		if !this.finishTask(task.Id, task.Version, err) {
			// 防止失败的任务无法重试
			break
		}
	}

	return nil
}

// 执行任务
func (this *Node) execTask(rpcClient *rpc.RPCClient, task *pb.NodeTask) error {
	var err error
	switch task.Type {
	case "ipItemChanged":
		err = this.execIPItemChangedTask()
	case "configChanged":
		err = this.execConfigChangedTask(task)
	case "nodeVersionChanged":
		err = this.execNodeVersionChangedTask()
	case "scriptsChanged":
		err = this.execScriptsChangedTask()
	case "nodeLevelChanged":
		err = this.execNodeLevelChangedTask(rpcClient)
	case "ddosProtectionChanged":
		err = this.execDDoSProtectionChangedTask(rpcClient)
	case "globalServerConfigChanged":
		err = this.execGlobalServerConfigChangedTask(rpcClient)
	case "userServersStateChanged":
		err = this.execUserServersStateChangedTask(rpcClient, task)
	case "uamPolicyChanged":
		err = this.execUAMPolicyChangedTask(rpcClient)
	case "httpCCPolicyChanged":
		err = this.execHTTPCCPolicyChangedTask(rpcClient)
	case "http3PolicyChanged":
		err = this.execHTTP3PolicyChangedTask(rpcClient)
	case "httpPagesPolicyChanged":
		err = this.execHTTPPagesPolicyChangedTask(rpcClient)
	case "updatingServers":
		err = this.execUpdatingServersTask(rpcClient)
	case "plusChanged":
		err = this.notifyPlusChange()
	case "toaChanged":
		err = this.execTOAChangedTask()
	default:
		remotelogs.Error("NODE", "task '"+types.String(task.Id)+"', type '"+task.Type+"' has not been handled")
	}

	return err
}

// 更新IP条目变更
func (this *Node) execIPItemChangedTask() error {
	// 防止阻塞
	select {
	case iplibrary.IPListUpdateNotify <- true:
	default:

	}
	return nil
}

// 更新节点配置变更
func (this *Node) execConfigChangedTask(task *pb.NodeTask) error {
	if task.ServerId > 0 {
		return this.syncServerConfig(task.ServerId)
	}
	if !task.IsPrimary {
		// 我们等等主节点配置准备完毕
		time.Sleep(2 * time.Second)
	}
	return this.syncConfig(task.Version)
}

// 节点程序版本号变更
func (this *Node) execNodeVersionChangedTask() error {
	if !sharedUpgradeManager.IsInstalling() {
		goman.New(func() {
			sharedUpgradeManager.Start()
		})
	}
	return nil
}

// 节点级别变更
func (this *Node) execNodeLevelChangedTask(rpcClient *rpc.RPCClient) error {
	levelInfoResp, err := rpcClient.NodeRPC.FindNodeLevelInfo(rpcClient.Context(), &pb.FindNodeLevelInfoRequest{})
	if err != nil {
		return err
	}

	if sharedNodeConfig != nil {
		sharedNodeConfig.Level = levelInfoResp.Level
	}

	var parentNodes = map[int64][]*nodeconfigs.ParentNodeConfig{}
	if len(levelInfoResp.ParentNodesMapJSON) > 0 {
		err = json.Unmarshal(levelInfoResp.ParentNodesMapJSON, &parentNodes)
		if err != nil {
			return errors.New("decode level info failed: " + err.Error())
		}
	}

	if sharedNodeConfig != nil {
		sharedNodeConfig.ParentNodes = parentNodes
	}

	return nil
}

// DDoS配置变更
func (this *Node) execDDoSProtectionChangedTask(rpcClient *rpc.RPCClient) error {
	resp, err := rpcClient.NodeRPC.FindNodeDDoSProtection(rpcClient.Context(), &pb.FindNodeDDoSProtectionRequest{})
	if err != nil {
		return err
	}
	if len(resp.DdosProtectionJSON) == 0 {
		if sharedNodeConfig != nil {
			sharedNodeConfig.DDoSProtection = nil
		}
		return nil
	}

	var ddosProtectionConfig = &ddosconfigs.ProtectionConfig{}
	err = json.Unmarshal(resp.DdosProtectionJSON, ddosProtectionConfig)
	if err != nil {
		return errors.New("decode DDoS protection config failed: " + err.Error())
	}

	if ddosProtectionConfig != nil && sharedNodeConfig != nil {
		sharedNodeConfig.DDoSProtection = ddosProtectionConfig
	}

	go func() {
		err = firewalls.SharedDDoSProtectionManager.Apply(ddosProtectionConfig)
		if err != nil {
			// 不阻塞
			remotelogs.Warn("NODE", "apply DDoS protection failed: "+err.Error())
		}
	}()

	return nil
}

// 服务全局配置变更
func (this *Node) execGlobalServerConfigChangedTask(rpcClient *rpc.RPCClient) error {
	resp, err := rpcClient.NodeRPC.FindNodeGlobalServerConfig(rpcClient.Context(), &pb.FindNodeGlobalServerConfigRequest{})
	if err != nil {
		return err
	}
	if len(resp.GlobalServerConfigJSON) > 0 {
		var globalServerConfig = serverconfigs.NewGlobalServerConfig()
		err = json.Unmarshal(resp.GlobalServerConfigJSON, globalServerConfig)
		if err != nil {
			return errors.New("decode global server config failed: " + err.Error())
		}

		if globalServerConfig != nil {
			err = globalServerConfig.Init()
			if err != nil {
				return errors.New("validate global server config failed: " + err.Error())
			}
			if sharedNodeConfig != nil {
				sharedNodeConfig.GlobalServerConfig = globalServerConfig
			}
		}
	}
	return nil
}

// 单个用户服务状态变更
func (this *Node) execUserServersStateChangedTask(rpcClient *rpc.RPCClient, task *pb.NodeTask) error {
	if task.UserId > 0 {
		resp, err := rpcClient.UserRPC.CheckUserServersState(rpcClient.Context(), &pb.CheckUserServersStateRequest{UserId: task.UserId})
		if err != nil {
			return err
		}

		SharedUserManager.UpdateUserServersIsEnabled(task.UserId, resp.IsEnabled)

		if resp.IsEnabled {
			err = this.syncUserServersConfig(task.UserId)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// 更新一组服务列表
func (this *Node) execUpdatingServersTask(rpcClient *rpc.RPCClient) error {
	if this.lastUpdatingServerListId <= 0 {
		this.lastUpdatingServerListId = sharedNodeConfig.UpdatingServerListId
	}

	resp, err := rpcClient.UpdatingServerListRPC.FindUpdatingServerLists(rpcClient.Context(), &pb.FindUpdatingServerListsRequest{LastId: this.lastUpdatingServerListId})
	if err != nil {
		return err
	}

	if resp.MaxId <= 0 || len(resp.ServersJSON) == 0 {
		return nil
	}

	var serverConfigs = []*serverconfigs.ServerConfig{}
	err = json.Unmarshal(resp.ServersJSON, &serverConfigs)
	if err != nil {
		return errors.New("decode server configs failed: " + err.Error())
	}

	if resp.MaxId > this.lastUpdatingServerListId {
		this.lastUpdatingServerListId = resp.MaxId
	}

	if len(serverConfigs) == 0 {
		return nil
	}

	this.locker.Lock()
	defer this.locker.Unlock()
	for _, serverConfig := range serverConfigs {
		if serverConfig == nil {
			continue
		}

		if serverConfig.IsOn {
			this.updatingServerMap[serverConfig.Id] = serverConfig
		} else {
			this.updatingServerMap[serverConfig.Id] = nil
		}
	}

	return nil
}

// 标记任务完成
func (this *Node) finishTask(taskId int64, taskVersion int64, taskErr error) (success bool) {
	if taskId <= 0 {
		return true
	}

	rpcClient, err := rpc.SharedRPC()
	if err != nil {
		remotelogs.Debug("NODE", "create rpc client failed: "+err.Error())
		return false
	}

	var isOk = taskErr == nil
	if isOk && taskVersion > this.lastTaskVersion {
		this.lastTaskVersion = taskVersion
	}

	var errMsg = ""
	if taskErr != nil {
		errMsg = taskErr.Error()
	}

	_, err = rpcClient.NodeTaskRPC.ReportNodeTaskDone(rpcClient.Context(), &pb.ReportNodeTaskDoneRequest{
		NodeTaskId: taskId,
		IsOk:       isOk,
		Error:      errMsg,
	})
	success = err == nil

	if err != nil {
		// 连接错误不需要上报到服务中心
		if rpc.IsConnError(err) {
			remotelogs.Debug("NODE", "report task done failed: "+err.Error())
		} else {
			remotelogs.Error("NODE", "report task done failed: "+err.Error())
		}
	}

	return success
}
