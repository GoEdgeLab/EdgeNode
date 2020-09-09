package main

import (
	"fmt"
	"github.com/TeaOSLab/EdgeNode/internal/apps"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/nodes"
	_ "github.com/iwind/TeaGo/bootstrap"
)

func main() {
	app := apps.NewAppCmd().
		Version(teaconst.Version).
		Product(teaconst.ProductName).
		Usage(teaconst.ProcessName + " [-v|start|stop|restart|sync|update]")

	app.On("sync", func() {
		// TODO
		fmt.Println("not implemented yet")
	})
	app.On("update", func() {
		// TODO
		fmt.Println("not implemented yet")
	})
	app.Run(func() {
		node := nodes.NewNode()
		node.Start()
	})
}
