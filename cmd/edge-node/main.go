package main

import (
	"fmt"
	"github.com/TeaOSLab/EdgeNode/internal/apps"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/nodes"
	"github.com/iwind/TeaGo/Tea"
	_ "github.com/iwind/TeaGo/bootstrap"
	"github.com/iwind/TeaGo/lists"
	"github.com/iwind/TeaGo/types"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func main() {
	app := apps.NewAppCmd().
		Version(teaconst.Version).
		Product(teaconst.ProductName).
		Usage(teaconst.ProcessName + " [-v|start|stop|restart|quit|test|daemon]")

	app.On("test", func() {
		err := nodes.NewNode().Test()
		if err != nil {
			_, _ = os.Stderr.WriteString(err.Error())
		}
	})
	app.On("daemon", func() {
		path := os.TempDir() + "/edge-node.sock"
		isDebug := lists.ContainsString(os.Args, "debug")
		isDebug = true
		for {
			conn, err := net.DialTimeout("unix", path, 1*time.Second)
			if err != nil {
				if isDebug {
					log.Println("[DAEMON]starting ...")
				}

				// 尝试启动
				err = func() error {
					exe, err := os.Executable()
					if err != nil {
						return err
					}
					cmd := exec.Command(exe)
					err = cmd.Start()
					if err != nil {
						return err
					}
					err = cmd.Wait()
					if err != nil {
						return err
					}
					return nil
				}()

				if err != nil {
					if isDebug {
						log.Println("[DAEMON]", err)
					}
					time.Sleep(1 * time.Second)
				} else {
					time.Sleep(5 * time.Second)
				}
			} else {
				_ = conn.Close()
				time.Sleep(5 * time.Second)
			}
		}
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
