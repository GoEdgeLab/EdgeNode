package nodes

import (
	"encoding/json"
	"errors"
	"github.com/TeaOSLab/EdgeCommon/pkg/nodeconfigs"
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeNode/internal/caches"
	"github.com/TeaOSLab/EdgeNode/internal/configs"
	"github.com/TeaOSLab/EdgeNode/internal/logs"
	"github.com/TeaOSLab/EdgeNode/internal/rpc"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/go-yaml/yaml"
	"github.com/iwind/TeaGo/Tea"
	tealogs "github.com/iwind/TeaGo/logs"
	"io/ioutil"
	"os"
	"runtime"
	"time"
)

var lastVersion = int64(-1)
var sharedNodeConfig *nodeconfigs.NodeConfig
var changeNotify = make(chan bool, 8)

// 节点
type Node struct {
}

func NewNode() *Node {
	return &Node{}
}

func (this *Node) Start() {
	// 读取API配置
	err := this.syncConfig(false)
	if err != nil {
		logs.Error("NODE", err.Error())
	}

	// 启动同步计时器
	this.startSyncTimer()

	// 状态变更计时器
	go NewNodeStatusExecutor().Listen()

	// 读取配置
	nodeConfig, err := nodeconfigs.SharedNodeConfig()
	if err != nil {
		logs.Error("NODE", "start failed: read node config failed: "+err.Error())
		return
	}
	err = nodeConfig.Init()
	if err != nil {
		logs.Error("NODE", "init node config failed: "+err.Error())
		return
	}
	sharedNodeConfig = nodeConfig

	// 设置rlimit
	_ = utils.SetRLimit(1024 * 1024)

	// 连接API
	go NewAPIStream().Start()

	// 启动端口
	err = sharedListenerManager.Start(nodeConfig)
	if err != nil {
		logs.Error("NODE", "start failed: "+err.Error())
	}

	// hold住进程
	select {}
}

// 读取API配置
func (this *Node) syncConfig(isFirstTime bool) error {
	// 检查api.yaml是否存在
	apiConfigFile := Tea.ConfigFile("api.yaml")
	_, err := os.Stat(apiConfigFile)
	if err != nil {
		if os.IsNotExist(err) {
			clusterErr := this.checkClusterConfig()
			if clusterErr != nil {
				if os.IsNotExist(clusterErr) {
					return err
				}
				return clusterErr
			}
		} else {
			return err
		}
	}

	rpcClient, err := rpc.SharedRPC()
	if err != nil {
		return errors.New("create rpc client failed: " + err.Error())
	}
	// TODO 这里考虑只同步版本号有变更的
	configResp, err := rpcClient.NodeRPC().ComposeNodeConfig(rpcClient.Context(), &pb.ComposeNodeConfigRequest{})
	if err != nil {
		return errors.New("read config from rpc failed: " + err.Error())
	}
	configJSON := configResp.NodeJSON
	nodeConfig := &nodeconfigs.NodeConfig{}
	err = json.Unmarshal(configJSON, nodeConfig)
	if err != nil {
		return errors.New("decode config failed: " + err.Error())
	}

	// 写入到文件中
	err = nodeConfig.Save()
	if err != nil {
		return err
	}

	// 如果版本相同，则只是保存
	if lastVersion == nodeConfig.Version {
		return nil
	}
	lastVersion = nodeConfig.Version

	err = nodeConfig.Init()
	if err != nil {
		return err
	}

	// max cpu
	if nodeConfig.MaxCPU > 0 && nodeConfig.MaxCPU < int32(runtime.NumCPU()) {
		runtime.GOMAXPROCS(int(nodeConfig.MaxCPU))
	} else {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}

	// 刷新配置
	if isFirstTime {
		logs.Println("NODE", "reloading config ...")
	} else {
		logs.Println("NODE", "loading config ...")
	}

	nodeconfigs.ResetNodeConfig(nodeConfig)
	caches.SharedManager.UpdatePolicies(nodeConfig.AllCachePolicies())
	sharedWAFManager.UpdatePolicies(nodeConfig.AllHTTPFirewallPolicies())
	sharedNodeConfig = nodeConfig

	if !isFirstTime {
		return sharedListenerManager.Start(nodeConfig)
	}

	return nil
}

// 启动同步计时器
func (this *Node) startSyncTimer() {
	// TODO 这个时间间隔可以自行设置
	ticker := time.NewTicker(60 * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				err := this.syncConfig(false)
				if err != nil {
					logs.Error("NODE", "sync config error: "+err.Error())
					continue
				}
			case <-changeNotify:
				err := this.syncConfig(false)
				if err != nil {
					logs.Error("NODE", "sync config error: "+err.Error())
					continue
				}
			}
		}
	}()
}

// 检查集群设置
func (this *Node) checkClusterConfig() error {
	configFile := Tea.ConfigFile("cluster.yaml")
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		return err
	}
	config := &configs.ClusterConfig{}
	err = yaml.Unmarshal(data, config)
	if err != nil {
		return err
	}

	rpcClient, err := rpc.NewRPCClient(&configs.APIConfig{
		RPC:    config.RPC,
		NodeId: config.ClusterId,
		Secret: config.Secret,
	})
	if err != nil {
		return err
	}

	tealogs.Println("[NODE]registering node ...")
	resp, err := rpcClient.NodeRPC().RegisterClusterNode(rpcClient.ClusterContext(config.ClusterId, config.Secret), &pb.RegisterClusterNodeRequest{Name: HOSTNAME})
	if err != nil {
		return err
	}
	tealogs.Println("[NODE]registered successfully")

	// 写入到配置文件中
	if len(resp.Endpoints) == 0 {
		resp.Endpoints = []string{}
	}
	apiConfig := &configs.APIConfig{
		RPC: struct {
			Endpoints []string `yaml:"endpoints"`
		}{
			Endpoints: resp.Endpoints,
		},
		NodeId: resp.UniqueId,
		Secret: resp.Secret,
	}
	tealogs.Println("[NODE]writing 'configs/api.yaml' ...")
	err = apiConfig.WriteFile(Tea.ConfigFile("api.yaml"))
	if err != nil {
		return err
	}
	tealogs.Println("[NODE]wrote 'configs/api.yaml' successfully")

	return nil
}
