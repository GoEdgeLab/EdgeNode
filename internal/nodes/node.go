package nodes

import (
	"encoding/json"
	"errors"
	"github.com/TeaOSLab/EdgeNode/internal/configs"
	"github.com/TeaOSLab/EdgeNode/internal/rpc"
	"github.com/TeaOSLab/EdgeNode/internal/rpc/pb"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/iwind/TeaGo/logs"
	"time"
)

var stop = make(chan bool)
var lastVersion = -1

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
	nodeConfig, err := configs.SharedNodeConfig()
	if err != nil {
		logs.Println("[NODE]start failed: read node config failed: " + err.Error())
		return
	}

	// 设置rlimit
	_ = utils.SetRLimit(1024 * 1024)

	// 启动端口
	err = sharedListenerManager.Start(nodeConfig)
	if err != nil {
		logs.Println("[NODE]start failed: " + err.Error())
	}

	// hold住进程
	<-stop
}

// 读取API配置
func (this *Node) syncConfig(isFirstTime bool) error {
	rpcClient, err := rpc.SharedRPC()
	if err != nil {
		return errors.New("[NODE]create rpc client failed: " + err.Error())
	}
	configResp, err := rpcClient.NodeRPC().ComposeNodeConfig(rpcClient.Context(), &pb.ComposeNodeConfigRequest{})
	if err != nil {
		return errors.New("[NODE]read config from rpc failed: " + err.Error())
	}
	configBytes := configResp.ConfigJSON
	nodeConfig := &configs.NodeConfig{}
	err = json.Unmarshal(configBytes, nodeConfig)
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

	// 刷新配置
	err = configs.ReloadNodeConfig()
	if err != nil {
		return err
	}

	if !isFirstTime {
		return sharedListenerManager.Start(nodeConfig)
	}

	return nil
}

// 启动同步计时器
func (this *Node) startSyncTimer() {
	ticker := time.NewTicker(60 * time.Second)
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
