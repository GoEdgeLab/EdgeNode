package nodes

import (
	"context"
	"crypto/tls"
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeNode/internal/configs"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/TeaOSLab/EdgeNode/internal/rpc"
	"github.com/TeaOSLab/EdgeNode/internal/trackers"
	"github.com/iwind/TeaGo/Tea"
	"github.com/iwind/TeaGo/logs"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

var sharedSyncAPINodesTask = NewSyncAPINodesTask()

func init() {
	events.On(events.EventStart, func() {
		goman.New(func() {
			sharedSyncAPINodesTask.Start()
		})
	})
	events.On(events.EventQuit, func() {
		sharedSyncAPINodesTask.Stop()
	})
}

// SyncAPINodesTask API节点同步任务
type SyncAPINodesTask struct {
	ticker *time.Ticker
}

func NewSyncAPINodesTask() *SyncAPINodesTask {
	return &SyncAPINodesTask{}
}

func (this *SyncAPINodesTask) Start() {
	this.ticker = time.NewTicker(5 * time.Minute)
	if Tea.IsTesting() {
		// 快速测试
		this.ticker = time.NewTicker(1 * time.Minute)
	}
	for range this.ticker.C {
		err := this.Loop()
		if err != nil {
			logs.Println("[TASK][SYNC_API_NODES_TASK]" + err.Error())
		}
	}
}

func (this *SyncAPINodesTask) Stop() {
	if this.ticker != nil {
		this.ticker.Stop()
	}
}

func (this *SyncAPINodesTask) Loop() error {
	config, err := configs.LoadAPIConfig()
	if err != nil {
		return err
	}

	// 是否禁止自动升级
	if config.RPC.DisableUpdate {
		return nil
	}

	var tr = trackers.Begin("SYNC_API_NODES")
	defer tr.End()

	// 获取所有可用的节点
	rpcClient, err := rpc.SharedRPC()
	if err != nil {
		return err
	}
	resp, err := rpcClient.APINodeRPC.FindAllEnabledAPINodes(rpcClient.Context(), &pb.FindAllEnabledAPINodesRequest{})
	if err != nil {
		return err
	}

	var newEndpoints = []string{}
	for _, node := range resp.ApiNodes {
		if !node.IsOn {
			continue
		}
		newEndpoints = append(newEndpoints, node.AccessAddrs...)
	}

	// 和现有的对比
	if this.isSame(newEndpoints, config.RPC.Endpoints) {
		return nil
	}

	// 测试是否有API节点可用
	var hasOk = this.testEndpoints(newEndpoints)
	if !hasOk {
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

func (this *SyncAPINodesTask) testEndpoints(endpoints []string) bool {
	if len(endpoints) == 0 {
		return false
	}

	var wg = sync.WaitGroup{}
	wg.Add(len(endpoints))

	var ok = false

	for _, endpoint := range endpoints {
		go func(endpoint string) {
			defer wg.Done()

			u, err := url.Parse(endpoint)
			if err != nil {
				return
			}

			ctx, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
			defer func() {
				cancelFunc()
			}()
			var conn *grpc.ClientConn
			if u.Scheme == "http" {
				conn, err = grpc.DialContext(ctx, u.Host, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
			} else if u.Scheme == "https" {
				conn, err = grpc.DialContext(ctx, u.Host, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
					InsecureSkipVerify: true,
				})), grpc.WithBlock())
			}
			if err != nil {
				return
			}
			_ = conn.Close()

			ok = true
		}(endpoint)
	}
	wg.Wait()

	return ok
}
