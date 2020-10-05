package nodes

import (
	"encoding/json"
	"errors"
	"github.com/TeaOSLab/EdgeCommon/pkg/nodeconfigs"
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeNode/internal/caches"
	"github.com/TeaOSLab/EdgeNode/internal/rpc"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/iwind/TeaGo/logs"
	"time"
)

var lastVersion = int64(-1)
var sharedNodeConfig *nodeconfigs.NodeConfig

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
		logs.Println(err.Error())
	}

	// 启动同步计时器
	this.startSyncTimer()

	// 状态变更计时器
	go NewNodeStatusExecutor().Listen()

	// 读取配置
	nodeConfig, err := nodeconfigs.SharedNodeConfig()
	if err != nil {
		logs.Println("[NODE]start failed: read node config failed: " + err.Error())
		return
	}
	err = nodeConfig.Init()
	if err != nil {
		logs.Println("[NODE]init node config failed: " + err.Error())
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
		logs.Println("[NODE]start failed: " + err.Error())
	}

	// hold住进程
	select {}
}

// 读取API配置
func (this *Node) syncConfig(isFirstTime bool) error {
	rpcClient, err := rpc.SharedRPC()
	if err != nil {
		return errors.New("[NODE]create rpc client failed: " + err.Error())
	}
	// TODO 这里考虑只同步版本号有变更的
	configResp, err := rpcClient.NodeRPC().ComposeNodeConfig(rpcClient.Context(), &pb.ComposeNodeConfigRequest{})
	if err != nil {
		return errors.New("[NODE]read config from rpc failed: " + err.Error())
	}
	configJSON := configResp.NodeJSON
	nodeConfig := &nodeconfigs.NodeConfig{}
	err = json.Unmarshal(configJSON, nodeConfig)
	if err != nil {
		return errors.New("[NODE]decode config failed: " + err.Error())
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

	// 刷新配置
	logs.Println("[NODE]reload config ...")
	nodeconfigs.ResetNodeConfig(nodeConfig)
	caches.SharedManager.UpdatePolicies(nodeConfig.AllCachePolicies())
	sharedNodeConfig = nodeConfig

	if !isFirstTime {
		return sharedListenerManager.Start(nodeConfig)
	}

	return nil
}

// 启动同步计时器
func (this *Node) startSyncTimer() {
	// TODO 这个时间间隔可以自行设置
	ticker := time.NewTicker(30 * time.Second)
	go func() {
		for range ticker.C {
			err := this.syncConfig(false)
			if err != nil {
				logs.Println("[NODE]sync config error: " + err.Error())
				continue
			}
		}
	}()
}
