package nodes

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeNode/internal/configs"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/rpc"
	"github.com/iwind/TeaGo/Tea"
	"github.com/iwind/TeaGo/logs"
	"sort"
	"strings"
	"time"
)

func init() {
	events.On(events.EventStart, func() {
		task := NewSyncAPINodesTask()
		go task.Start()
	})
}

// API节点同步任务
type SyncAPINodesTask struct {
}

func NewSyncAPINodesTask() *SyncAPINodesTask {
	return &SyncAPINodesTask{}
}

func (this *SyncAPINodesTask) Start() {
	ticker := time.NewTicker(5 * time.Minute)
	if Tea.IsTesting() {
		// 快速测试
		ticker = time.NewTicker(1 * time.Minute)
	}
	events.On(events.EventQuit, func() {
		remotelogs.Println("SYNC_API_NODES_TASK", "quit task")
		ticker.Stop()
	})
	for range ticker.C {
		err := this.Loop()
		if err != nil {
			logs.Println("[TASK][SYNC_API_NODES_TASK]" + err.Error())
		}
	}
}

func (this *SyncAPINodesTask) Loop() error {
	// 获取所有可用的节点
	rpcClient, err := rpc.SharedRPC()
	if err != nil {
		return err
	}
	resp, err := rpcClient.APINodeRPC().FindAllEnabledAPINodes(rpcClient.Context(), &pb.FindAllEnabledAPINodesRequest{})
	if err != nil {
		return err
	}

	newEndpoints := []string{}
	for _, node := range resp.Nodes {
		if !node.IsOn {
			continue
		}
		newEndpoints = append(newEndpoints, node.AccessAddrs...)
	}

	// 和现有的对比
	config, err := configs.LoadAPIConfig()
	if err != nil {
		return err
	}
	if this.isSame(newEndpoints, config.RPC.Endpoints) {
		return nil
	}

	// 修改RPC对象配置
	config.RPC.Endpoints = newEndpoints
	err = rpcClient.UpdateConfig(config)
	if err != nil {
		return err
	}

	// 保存到文件
	err = config.WriteFile(Tea.ConfigFile("api.yaml"))
	if err != nil {
		return err
	}

	return nil
}

func (this *SyncAPINodesTask) isSame(endpoints1 []string, endpoints2 []string) bool {
	sort.Strings(endpoints1)
	sort.Strings(endpoints2)
	return strings.Join(endpoints1, "&") == strings.Join(endpoints2, "&")
}
