package nodes

import (
	"github.com/TeaOSLab/EdgeNode/internal/configs"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/iwind/TeaGo/logs"
)

var sharedNodeConfig *configs.NodeConfig = nil
var stop = make(chan bool)

type Node struct {
}

func NewNode() *Node {
	return &Node{}
}

func (this *Node) Start() {
	// 读取配置
	nodeConfig, err := configs.SharedNodeConfig()
	if err != nil {
		logs.Println("[NODE]start failed: read node config failed: " + err.Error())
		return
	}
	sharedNodeConfig = nodeConfig

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
