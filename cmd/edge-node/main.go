package main

import (
	"github.com/TeaOSLab/EdgeNode/internal/nodes"
	_ "github.com/iwind/TeaGo/bootstrap"
)

func main() {
	node := nodes.NewNode()
	node.Start()
}
