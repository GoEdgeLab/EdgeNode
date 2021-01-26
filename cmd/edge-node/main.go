package main

import (
	"fmt"
	"github.com/TeaOSLab/EdgeNode/internal/apps"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/nodes"
	"github.com/iwind/TeaGo/Tea"
	_ "github.com/iwind/TeaGo/bootstrap"
	"github.com/iwind/TeaGo/types"
	"io/ioutil"
	"os"
	"syscall"
)

func main() {
	app := apps.NewAppCmd().
		Version(teaconst.Version).
		Product(teaconst.ProductName).
		Usage(teaconst.ProcessName + " [-v|start|stop|restart|quit|test|service|daemon]")

	app.On("test", func() {
		err := nodes.NewNode().Test()
		if err != nil {
			_, _ = os.Stderr.WriteString(err.Error())
		}
	})
	app.On("daemon", func() {
		nodes.NewNode().Daemon()
	})
	app.On("service", func() {
		err := nodes.NewNode().InstallSystemService()
		if err != nil {
			fmt.Println("[ERROR]install failed: " + err.Error())
			return
		}
		fmt.Println("done")
	})
	app.On("quit", func() {
		pidFile := Tea.Root + "/bin/pid"
		data, err := ioutil.ReadFile(pidFile)
		if err != nil {
			fmt.Println("[ERROR]quit failed: " + err.Error())
			return
		}
		pid := types.Int(string(data))
		if pid == 0 {
			fmt.Println("[ERROR]quit failed: pid=0")
			return
		}

		process, err := os.FindProcess(pid)
		if err != nil {
			return
		}
		if process != nil {
			_ = process.Signal(syscall.SIGQUIT)
		}
	})
	app.Run(func() {
		node := nodes.NewNode()
		node.Start()
	})
}
