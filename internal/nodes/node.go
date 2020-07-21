package nodes

import (
	"github.com/TeaOSLab/EdgeNode/internal/configs"
	"github.com/iwind/TeaGo/logs"
)

var sharedNodeConfig *configs.NodeConfig = nil

type Node struct {
}

func NewNode() *Node {
	return &Node{}
}

func (this *Node) Start() {
	nodeConfig, err := configs.SharedNodeConfig()
	if err != nil {
		logs.Println("[NODE]start failed: read node config failed: " + err.Error())
		return
	}
	sharedNodeConfig = nodeConfig

	logs.PrintAsJSON(nodeConfig)
}
