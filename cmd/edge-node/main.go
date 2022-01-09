package main

import (
	"encoding/json"
	"fmt"
	"github.com/TeaOSLab/EdgeNode/internal/apps"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/nodes"
	_ "github.com/iwind/TeaGo/bootstrap"
	"github.com/iwind/TeaGo/logs"
	"github.com/iwind/TeaGo/maps"
	"github.com/iwind/TeaGo/types"
	"github.com/iwind/gosock/pkg/gosock"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"sort"
)

func main() {
	app := apps.NewAppCmd().
		Version(teaconst.Version).
		Product(teaconst.ProductName).
		Usage(teaconst.ProcessName + " [-v|start|stop|restart|status|quit|test|service|daemon|pprof]")

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
		var sock = gosock.NewTmpSock(teaconst.ProcessName)
		_, err := sock.Send(&gosock.Command{Code: "quit"})
		if err != nil {
			fmt.Println("[ERROR]quit failed: " + err.Error())
			return
		}
		fmt.Println("done")
	})
	app.On("pprof", func() {
		// TODO 自己指定端口
		addr := "127.0.0.1:6060"
		logs.Println("starting with pprof '" + addr + "'...")

		go func() {
			err := http.ListenAndServe(addr, nil)
			if err != nil {
				logs.Println("[error]" + err.Error())
			}
		}()

		node := nodes.NewNode()
		node.Start()
	})
	app.On("trackers", func() {
		var sock = gosock.NewTmpSock(teaconst.ProcessName)
		reply, err := sock.Send(&gosock.Command{Code: "trackers"})
		if err != nil {
			fmt.Println("[ERROR]" + err.Error())
		} else {
			labelsMap, ok := reply.Params["labels"]
			if ok {
				labels, ok := labelsMap.(map[string]interface{})
				if ok {
					if len(labels) == 0 {
						fmt.Println("no labels yet")
					} else {
						var labelNames = []string{}
						for label := range labels {
							labelNames = append(labelNames, label)
						}
						sort.Strings(labelNames)

						for _, labelName := range labelNames {
							fmt.Println(labelName + ": " + fmt.Sprintf("%.6f", labels[labelName]))
						}
					}
				}
			}
		}
	})
	app.On("goman", func() {
		var sock = gosock.NewTmpSock(teaconst.ProcessName)
		reply, err := sock.Send(&gosock.Command{Code: "goman"})
		if err != nil {
			fmt.Println("[ERROR]" + err.Error())
		} else {
			instancesJSON, err := json.MarshalIndent(reply.Params, "", "  ")
			if err != nil {
				fmt.Println("[ERROR]" + err.Error())
			} else {
				fmt.Println(string(instancesJSON))
			}
		}
	})
	app.On("conns", func() {
		var sock = gosock.NewTmpSock(teaconst.ProcessName)
		reply, err := sock.Send(&gosock.Command{Code: "conns"})
		if err != nil {
			fmt.Println("[ERROR]" + err.Error())
		} else {
			resultJSON, err := json.MarshalIndent(reply.Params, "", "  ")
			if err != nil {
				fmt.Println("[ERROR]" + err.Error())
			} else {
				fmt.Println(string(resultJSON))
			}
		}
	})
	app.On("gc", func() {
		var sock = gosock.NewTmpSock(teaconst.ProcessName)
		_, err := sock.Send(&gosock.Command{Code: "gc"})
		if err != nil {
			fmt.Println("[ERROR]" + err.Error())
		} else {
			fmt.Println("ok")
		}
	})
	app.On("ip.drop", func() {
		var args = os.Args[2:]
		if len(args) == 0 {
			fmt.Println("Usage: edge-node ip.drop IP [--timeout=SECONDS]")
			return
		}
		var ip = args[0]
		if len(net.ParseIP(ip)) == 0 {
			fmt.Println("IP '" + ip + "' is invalid")
			return
		}
		var timeoutSeconds = 0
		var options = app.ParseOptions(args[1:])
		timeout, ok := options["timeout"]
		if ok {
			timeoutSeconds = types.Int(timeout[0])
		}

		fmt.Println("drop ip '" + ip + "' for '" + types.String(timeoutSeconds) + "' seconds")
		var sock = gosock.NewTmpSock(teaconst.ProcessName)
		reply, err := sock.Send(&gosock.Command{
			Code: "dropIP",
			Params: map[string]interface{}{
				"ip":             ip,
				"timeoutSeconds": timeoutSeconds,
			},
		})
		if err != nil {
			fmt.Println("[ERROR]" + err.Error())
		} else {
			var errString = maps.NewMap(reply.Params).GetString("error")
			if len(errString) > 0 {
				fmt.Println("[ERROR]" + errString)
			} else {
				fmt.Println("ok")
			}
		}
	})
	app.On("ip.reject", func() {
		var args = os.Args[2:]
		if len(args) == 0 {
			fmt.Println("Usage: edge-node ip.reject IP [--timeout=SECONDS]")
			return
		}
		var ip = args[0]
		if len(net.ParseIP(ip)) == 0 {
			fmt.Println("IP '" + ip + "' is invalid")
			return
		}
		var timeoutSeconds = 0
		var options = app.ParseOptions(args[1:])
		timeout, ok := options["timeout"]
		if ok {
			timeoutSeconds = types.Int(timeout[0])
		}

		fmt.Println("reject ip '" + ip + "' for '" + types.String(timeoutSeconds) + "' seconds")

		var sock = gosock.NewTmpSock(teaconst.ProcessName)
		reply, err := sock.Send(&gosock.Command{
			Code: "rejectIP",
			Params: map[string]interface{}{
				"ip":             ip,
				"timeoutSeconds": timeoutSeconds,
			},
		})
		if err != nil {
			fmt.Println("[ERROR]" + err.Error())
		} else {
			var errString = maps.NewMap(reply.Params).GetString("error")
			if len(errString) > 0 {
				fmt.Println("[ERROR]" + errString)
			} else {
				fmt.Println("ok")
			}
		}
	})
	app.On("ip.remove", func() {
		var args = os.Args[2:]
		if len(args) == 0 {
			fmt.Println("Usage: edge-node ip.remove IP")
			return
		}
		var ip = args[0]
		if len(net.ParseIP(ip)) == 0 {
			fmt.Println("IP '" + ip + "' is invalid")
			return
		}

		var sock = gosock.NewTmpSock(teaconst.ProcessName)
		reply, err := sock.Send(&gosock.Command{
			Code: "removeIP",
			Params: map[string]interface{}{
				"ip": ip,
			},
		})
		if err != nil {
			fmt.Println("[ERROR]" + err.Error())
		} else {
			var errString = maps.NewMap(reply.Params).GetString("error")
			if len(errString) > 0 {
				fmt.Println("[ERROR]" + errString)
			} else {
				fmt.Println("ok")
			}
		}
	})
	app.Run(func() {
		node := nodes.NewNode()
		node.Start()
	})
}
