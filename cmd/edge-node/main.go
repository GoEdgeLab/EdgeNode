package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/TeaOSLab/EdgeNode/internal/apps"
	"github.com/TeaOSLab/EdgeNode/internal/configs"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/nodes"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	fsutils "github.com/TeaOSLab/EdgeNode/internal/utils/fs"
	"github.com/iwind/TeaGo/Tea"
	_ "github.com/iwind/TeaGo/bootstrap"
	"github.com/iwind/TeaGo/logs"
	"github.com/iwind/TeaGo/maps"
	"github.com/iwind/TeaGo/types"
	"github.com/iwind/gosock/pkg/gosock"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func main() {
	var app = apps.NewAppCmd().
		Version(teaconst.Version).
		Product(teaconst.ProductName).
		Usage(teaconst.ProcessName + " [-v|start|stop|restart|status|quit|test|reload|service|daemon|pprof|accesslog|uninstall]").
		Usage(teaconst.ProcessName + " [trackers|goman|conns|gc|bandwidth|disk|cache.garbage]").
		Usage(teaconst.ProcessName + " [ip.drop|ip.reject|ip.remove|ip.close] IP")

	app.On("start:before", func() {
		// validate config
		_, err := configs.LoadAPIConfig()
		if err != nil {
			// validate cluster config
			_, clusterErr := configs.LoadClusterConfig()
			if clusterErr != nil { // fail again
				fmt.Println("[ERROR]start failed: load api config from '" + Tea.ConfigFile(configs.ConfigFileName) + "' failed: " + err.Error())
				os.Exit(0)
			}
		}
	})
	app.On("uninstall", func() {
		// service
		fmt.Println("Uninstall service ...")
		var manager = utils.NewServiceManager(teaconst.ProcessName, teaconst.ProductName)
		go func() {
			_ = manager.Uninstall()
		}()

		// stop
		fmt.Println("Stopping ...")
		_, _ = gosock.NewTmpSock(teaconst.ProcessName).SendTimeout(&gosock.Command{Code: "stop"}, 1*time.Second)

		// delete files
		var exe, _ = os.Executable()
		if len(exe) == 0 {
			return
		}

		var dir = filepath.Dir(filepath.Dir(exe)) // ROOT / bin / exe

		// verify dir
		{
			fmt.Println("Checking '" + dir + "' ...")
			for _, subDir := range []string{"bin/" + filepath.Base(exe), "configs", "logs"} {
				_, err := os.Stat(dir + "/" + subDir)
				if err != nil {
					fmt.Println("[ERROR]program directory structure has been broken, please remove it manually.")
					return
				}
			}

			fmt.Println("Removing '" + dir + "' ...")
			err := os.RemoveAll(dir)
			if err != nil {
				fmt.Println("[ERROR]remove failed: " + err.Error())
			}
		}

		// delete symbolic links
		fmt.Println("Removing symbolic links ...")
		_ = os.Remove("/usr/bin/" + teaconst.ProcessName)
		_ = os.Remove("/var/log/" + teaconst.ProcessName)

		// delete configs
		// nothing to delete for EdgeNode

		// delete sock
		fmt.Println("Removing temporary files ...")
		var tempDir = os.TempDir()
		_ = os.Remove(tempDir + "/" + teaconst.ProcessName + ".sock")
		_ = os.Remove(tempDir + "/" + teaconst.AccessLogSockName)

		// cache ...
		fmt.Println("Please delete cache directories by yourself.")

		// done
		fmt.Println("[DONE]")
	})
	app.On("test", func() {
		err := nodes.NewNode().Test()
		if err != nil {
			_, _ = os.Stderr.WriteString(err.Error())
		}
	})
	app.On("reload", func() {
		var sock = gosock.NewTmpSock(teaconst.ProcessName)
		reply, err := sock.Send(&gosock.Command{Code: "reload"})
		if err != nil {
			fmt.Println("[ERROR]" + err.Error())
		} else {
			var params = maps.NewMap(reply.Params)
			if params.Has("error") {
				fmt.Println("[ERROR]" + params.GetString("error"))
			} else {
				fmt.Println("ok")
			}
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
		var flagSet = flag.NewFlagSet("pprof", flag.ExitOnError)
		var addr string
		flagSet.StringVar(&addr, "addr", "", "")
		_ = flagSet.Parse(os.Args[2:])

		if len(addr) == 0 {
			addr = "127.0.0.1:6060"
		}
		logs.Println("starting with pprof '" + addr + "'...")

		go func() {
			err := http.ListenAndServe(addr, nil)
			if err != nil {
				logs.Println("[ERROR]" + err.Error())
			}
		}()

		var node = nodes.NewNode()
		node.Start()
	})
	app.On("dbstat", func() {
		teaconst.EnableDBStat = true

		var node = nodes.NewNode()
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
			fmt.Println("Usage: edge-node ip.drop IP [--timeout=SECONDS] [--async]")
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
		var async = false
		_, ok = options["async"]
		if ok {
			async = true
		}

		fmt.Println("drop ip '" + ip + "' for '" + types.String(timeoutSeconds) + "' seconds")
		var sock = gosock.NewTmpSock(teaconst.ProcessName)
		reply, err := sock.Send(&gosock.Command{
			Code: "dropIP",
			Params: map[string]interface{}{
				"ip":             ip,
				"timeoutSeconds": timeoutSeconds,
				"async":          async,
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
	app.On("ip.close", func() {
		var args = os.Args[2:]
		if len(args) == 0 {
			fmt.Println("Usage: edge-node ip.close IP")
			return
		}
		var ip = args[0]
		if len(net.ParseIP(ip)) == 0 {
			fmt.Println("IP '" + ip + "' is invalid")
			return
		}

		fmt.Println("close ip '" + ip)

		var sock = gosock.NewTmpSock(teaconst.ProcessName)
		reply, err := sock.Send(&gosock.Command{
			Code: "closeIP",
			Params: map[string]any{
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
	app.On("accesslog", func() {
		// local sock
		var tmpDir = os.TempDir()
		var sockFile = tmpDir + "/" + teaconst.AccessLogSockName
		_, err := os.Stat(sockFile)
		if err != nil {
			if !os.IsNotExist(err) {
				fmt.Println("[ERROR]" + err.Error())
				return
			}
		}

		var processSock = gosock.NewTmpSock(teaconst.ProcessName)
		reply, err := processSock.Send(&gosock.Command{
			Code: "accesslog",
		})
		if err != nil {
			fmt.Println("[ERROR]" + err.Error())
			return
		}
		if reply.Code == "error" {
			var errString = maps.NewMap(reply.Params).GetString("error")
			if len(errString) > 0 {
				fmt.Println("[ERROR]" + errString)
				return
			}
		}

		conn, err := net.Dial("unix", sockFile)
		if err != nil {
			fmt.Println("[ERROR]start reading access log failed: " + err.Error())
			return
		}
		defer func() {
			_ = conn.Close()
		}()
		var buf = make([]byte, 1024)
		for {
			n, err := conn.Read(buf)
			if n > 0 {
				fmt.Print(string(buf[:n]))
			}
			if err != nil {
				break
			}
		}
	})
	app.On("bandwidth", func() {
		var sock = gosock.NewTmpSock(teaconst.ProcessName)
		reply, err := sock.Send(&gosock.Command{Code: "bandwidth"})
		if err != nil {
			fmt.Println("[ERROR]" + err.Error())
			return
		}
		var statsMap = maps.NewMap(reply.Params).Get("stats")
		statsJSON, err := json.MarshalIndent(statsMap, "", "  ")
		if err != nil {
			fmt.Println("[ERROR]" + err.Error())
			return
		}
		fmt.Println(string(statsJSON))
	})
	app.On("disk", func() {
		var args = os.Args[2:]
		if len(args) > 0 {
			switch args[0] {
			case "speed":
				speedMB, isFast, err := fsutils.CheckDiskIsFast()
				if err != nil {
					fmt.Println("[ERROR]" + err.Error())
				} else {
					fmt.Printf("Speed: %.0fMB/s\n", speedMB)
					if isFast {
						fmt.Println("IsFast: true")
					} else {
						fmt.Println("IsFast: false")
					}
				}
			default:
				fmt.Println("Usage: edge-node disk [speed]")
			}
		} else {
			fmt.Println("Usage: edge-node disk [speed]")
		}
	})
	app.On("cache.garbage", func() {
		fmt.Println("scanning ...")

		var shouldDelete bool
		for _, arg := range os.Args {
			if strings.TrimLeft(arg, "-") == "delete" {
				shouldDelete = true
			}
		}

		var progressSock = gosock.NewTmpSock(teaconst.CacheGarbageSockName)
		progressSock.OnCommand(func(cmd *gosock.Command) {
			var params = maps.NewMap(cmd.Params)
			if cmd.Code == "progress" {
				fmt.Printf("%.2f%% %d\n", params.GetFloat64("progress")*100, params.GetInt("count"))
				_ = cmd.ReplyOk()
			}
		})
		go func() {
			_ = progressSock.Listen()
		}()
		time.Sleep(1 * time.Second)

		var sock = gosock.NewTmpSock(teaconst.ProcessName)
		reply, err := sock.Send(&gosock.Command{
			Code:   "cache.garbage",
			Params: map[string]any{"delete": shouldDelete},
		})
		if err != nil {
			fmt.Println("[ERROR]" + err.Error())
			return
		}

		var params = maps.NewMap(reply.Params)
		if params.GetBool("isOk") {
			var count = params.GetInt("count")
			fmt.Println("found", count, "bad caches")

			if count > 0 {
				fmt.Println("======")
				var sampleFiles = params.GetSlice("sampleFiles")
				for _, file := range sampleFiles {
					fmt.Println(types.String(file))
				}
				if count > len(sampleFiles) {
					fmt.Println("... more files")
				}
			}
		} else {
			fmt.Println("[ERROR]" + params.GetString("error"))
		}
	})
	app.Run(func() {
		var node = nodes.NewNode()
		node.Start()
	})
}
