package nodes

import (
	"encoding/json"
	"errors"
	"github.com/TeaOSLab/EdgeCommon/pkg/nodeconfigs"
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/apps"
	"github.com/TeaOSLab/EdgeNode/internal/caches"
	"github.com/TeaOSLab/EdgeNode/internal/configs"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/iplibrary"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/rpc"
	"github.com/TeaOSLab/EdgeNode/internal/stats"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/go-yaml/yaml"
	"github.com/iwind/TeaGo/Tea"
	"github.com/iwind/TeaGo/lists"
	"github.com/iwind/TeaGo/logs"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

var sharedNodeConfig *nodeconfigs.NodeConfig
var nodeTaskNotify = make(chan bool, 8)
var DaemonIsOn = false
var DaemonPid = 0

// Node 节点
type Node struct {
	isLoaded bool
}

func NewNode() *Node {
	return &Node{}
}

// Test 检查配置
func (this *Node) Test() error {
	// 检查是否能连接API
	rpcClient, err := rpc.SharedRPC()
	if err != nil {
		return errors.New("test rpc failed: " + err.Error())
	}
	_, err = rpcClient.APINodeRPC().FindCurrentAPINodeVersion(rpcClient.Context(), &pb.FindCurrentAPINodeVersionRequest{})
	if err != nil {
		return errors.New("test rpc failed: " + err.Error())
	}

	return nil
}

// Start 启动
func (this *Node) Start() {
	_, ok := os.LookupEnv("EdgeDaemon")
	if ok {
		remotelogs.Println("NODE", "start from daemon")
		DaemonIsOn = true
		DaemonPid = os.Getppid()
	}

	// 启动事件
	events.Notify(events.EventStart)

	// 处理信号
	this.listenSignals()

	// 本地Sock
	err := this.listenSock()
	if err != nil {
		remotelogs.Error("NODE", err.Error())
		return
	}

	// 读取API配置
	tryTimes := 0
	for {
		err = this.syncConfig()
		if err != nil {
			tryTimes++

			if tryTimes%10 == 0 {
				remotelogs.Error("NODE", err.Error())
			}
			time.Sleep(1 * time.Second)

			// 不做长时间的无意义的重试
			if tryTimes > 1000 {
				return
			}
		} else {
			break
		}
	}

	// 启动同步计时器
	this.startSyncTimer()

	// 状态变更计时器
	go NewNodeStatusExecutor().Listen()

	// 读取配置
	nodeConfig, err := nodeconfigs.SharedNodeConfig()
	if err != nil {
		remotelogs.Error("NODE", "start failed: read node config failed: "+err.Error())
		return
	}
	err = nodeConfig.Init()
	if err != nil {
		remotelogs.Error("NODE", "init node config failed: "+err.Error())
		return
	}
	sharedNodeConfig = nodeConfig

	// 发送事件
	events.Notify(events.EventLoaded)

	// 设置rlimit
	_ = utils.SetRLimit(1024 * 1024)

	// 连接API
	go NewAPIStream().Start()

	// 统计
	go stats.SharedTrafficStatManager.Start(func() *nodeconfigs.NodeConfig {
		return sharedNodeConfig
	})
	go stats.SharedHTTPRequestStatManager.Start()

	// 启动端口
	err = sharedListenerManager.Start(nodeConfig)
	if err != nil {
		remotelogs.Error("NODE", "start failed: "+err.Error())
		return
	}

	// 写入PID
	err = apps.WritePid()
	if err != nil {
		remotelogs.Error("NODE", "write pid failed: "+err.Error())
		return
	}

	// hold住进程
	select {}
}

// Daemon 实现守护进程
func (this *Node) Daemon() {
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

				// 可以标记当前是从守护进程启动的
				_ = os.Setenv("EdgeDaemon", "on")

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
}

// InstallSystemService 安装系统服务
func (this *Node) InstallSystemService() error {
	shortName := teaconst.SystemdServiceName

	exe, err := os.Executable()
	if err != nil {
		return err
	}

	manager := utils.NewServiceManager(shortName, teaconst.ProductName)
	err = manager.Install(exe, []string{})
	if err != nil {
		return err
	}
	return nil
}

// 处理信号
func (this *Node) listenSignals() {
	signals := make(chan os.Signal)
	signal.Notify(signals, syscall.SIGQUIT)
	go func() {
		for s := range signals {
			switch s {
			case syscall.SIGQUIT:
				events.Notify(events.EventQuit)

				// 监控连接数，如果连接数为0，则退出进程
				go func() {
					for {
						countActiveConnections := sharedListenerManager.TotalActiveConnections()
						if countActiveConnections <= 0 {
							os.Exit(0)
							return
						}
						time.Sleep(1 * time.Second)
					}
				}()
			}
		}
	}()
}

// 循环
func (this *Node) loop() error {
	// 检查api.yaml是否存在
	apiConfigFile := Tea.ConfigFile("api.yaml")
	_, err := os.Stat(apiConfigFile)
	if err != nil {
		return nil
	}

	rpcClient, err := rpc.SharedRPC()
	if err != nil {
		return errors.New("create rpc client failed: " + err.Error())
	}

	nodeCtx := rpcClient.Context()
	tasksResp, err := rpcClient.NodeTaskRPC().FindNodeTasks(nodeCtx, &pb.FindNodeTasksRequest{})
	if err != nil {
		return errors.New("read node tasks failed: " + err.Error())
	}
	for _, task := range tasksResp.NodeTasks {
		switch task.Type {
		case "ipItemChanged":
			iplibrary.IPListUpdateNotify <- true

			// 修改为已同步
			_, err = rpcClient.NodeTaskRPC().ReportNodeTaskDone(nodeCtx, &pb.ReportNodeTaskDoneRequest{
				NodeTaskId: task.Id,
				IsOk:       true,
				Error:      "",
			})
			if err != nil {
				return err
			}
		case "configChanged":
			err := this.syncConfig()
			if err != nil {
				_, err = rpcClient.NodeTaskRPC().ReportNodeTaskDone(nodeCtx, &pb.ReportNodeTaskDoneRequest{
					NodeTaskId: task.Id,
					IsOk:       false,
					Error:      err.Error(),
				})
			} else {
				_, err = rpcClient.NodeTaskRPC().ReportNodeTaskDone(nodeCtx, &pb.ReportNodeTaskDoneRequest{
					NodeTaskId: task.Id,
					IsOk:       true,
					Error:      "",
				})
			}
			if err != nil {
				return err
			}
		case "nodeVersionChanged":
			go sharedUpgradeManager.Start()
		}
	}

	return nil
}

// 读取API配置
func (this *Node) syncConfig() error {
	// 检查api.yaml是否存在
	apiConfigFile := Tea.ConfigFile("api.yaml")
	_, err := os.Stat(apiConfigFile)
	if err != nil {
		if os.IsNotExist(err) {
			clusterErr := this.checkClusterConfig()
			if clusterErr != nil {
				if os.IsNotExist(clusterErr) {
					return err
				}
				return clusterErr
			}
		} else {
			return err
		}
	}

	rpcClient, err := rpc.SharedRPC()
	if err != nil {
		return errors.New("create rpc client failed: " + err.Error())
	}

	// 获取同步任务
	nodeCtx := rpcClient.Context()

	// TODO 这里考虑只同步版本号有变更的
	configResp, err := rpcClient.NodeRPC().FindCurrentNodeConfig(nodeCtx, &pb.FindCurrentNodeConfigRequest{
		Version: -1, // 更新所有版本
	})
	if err != nil {
		return errors.New("read config from rpc failed: " + err.Error())
	}
	if !configResp.IsChanged {
		return nil
	}

	configJSON := configResp.NodeJSON
	nodeConfig := &nodeconfigs.NodeConfig{}
	err = json.Unmarshal(configJSON, nodeConfig)
	if err != nil {
		return errors.New("decode config failed: " + err.Error())
	}

	// 写入到文件中
	err = nodeConfig.Save()
	if err != nil {
		return err
	}

	err = nodeConfig.Init()
	if err != nil {
		return err
	}

	// max cpu
	if nodeConfig.MaxCPU > 0 && nodeConfig.MaxCPU < int32(runtime.NumCPU()) {
		runtime.GOMAXPROCS(int(nodeConfig.MaxCPU))
	} else {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}

	// 刷新配置
	if this.isLoaded {
		remotelogs.Println("NODE", "reloading config ...")
	} else {
		remotelogs.Println("NODE", "loading config ...")
	}

	nodeconfigs.ResetNodeConfig(nodeConfig)
	caches.SharedManager.MaxDiskCapacity = nodeConfig.MaxCacheDiskCapacity
	caches.SharedManager.MaxMemoryCapacity = nodeConfig.MaxCacheMemoryCapacity
	if nodeConfig.HTTPCachePolicy != nil {
		caches.SharedManager.UpdatePolicies([]*serverconfigs.HTTPCachePolicy{nodeConfig.HTTPCachePolicy})
	} else {
		caches.SharedManager.UpdatePolicies([]*serverconfigs.HTTPCachePolicy{})
	}

	sharedWAFManager.UpdatePolicies(nodeConfig.FindAllFirewallPolicies())
	iplibrary.SharedActionManager.UpdateActions(nodeConfig.FirewallActions)
	sharedNodeConfig = nodeConfig

	// 发送事件
	events.Notify(events.EventReload)

	if this.isLoaded {
		return sharedListenerManager.Start(nodeConfig)
	}

	this.isLoaded = true

	return nil
}

// 启动同步计时器
func (this *Node) startSyncTimer() {
	// TODO 这个时间间隔可以自行设置
	ticker := time.NewTicker(60 * time.Second)
	events.On(events.EventQuit, func() {
		remotelogs.Println("NODE", "quit sync timer")
		ticker.Stop()
	})
	go func() {
		for {
			select {
			case <-ticker.C:
				err := this.loop()
				if err != nil {
					remotelogs.Error("NODE", "sync config error: "+err.Error())
					continue
				}
			case <-nodeTaskNotify:
				err := this.loop()
				if err != nil {
					remotelogs.Error("NODE", "sync config error: "+err.Error())
					continue
				}
			}
		}
	}()
}

// 检查集群设置
func (this *Node) checkClusterConfig() error {
	configFile := Tea.ConfigFile("cluster.yaml")
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		return err
	}
	config := &configs.ClusterConfig{}
	err = yaml.Unmarshal(data, config)
	if err != nil {
		return err
	}

	rpcClient, err := rpc.NewRPCClient(&configs.APIConfig{
		RPC:    config.RPC,
		NodeId: config.ClusterId,
		Secret: config.Secret,
	})
	if err != nil {
		return err
	}

	logs.Println("[NODE]registering node ...")
	resp, err := rpcClient.NodeRPC().RegisterClusterNode(rpcClient.ClusterContext(config.ClusterId, config.Secret), &pb.RegisterClusterNodeRequest{Name: HOSTNAME})
	if err != nil {
		return err
	}
	logs.Println("[NODE]registered successfully")

	// 写入到配置文件中
	if len(resp.Endpoints) == 0 {
		resp.Endpoints = []string{}
	}
	apiConfig := &configs.APIConfig{
		RPC: struct {
			Endpoints []string `yaml:"endpoints"`
		}{
			Endpoints: resp.Endpoints,
		},
		NodeId: resp.UniqueId,
		Secret: resp.Secret,
	}
	logs.Println("[NODE]writing 'configs/api.yaml' ...")
	err = apiConfig.WriteFile(Tea.ConfigFile("api.yaml"))
	if err != nil {
		return err
	}
	logs.Println("[NODE]wrote 'configs/api.yaml' successfully")

	return nil
}

// 监听本地sock
func (this *Node) listenSock() error {
	path := os.TempDir() + "/edge-node.sock"

	// 检查是否已经存在
	_, err := os.Stat(path)
	if err == nil {
		conn, err := net.Dial("unix", path)
		if err != nil {
			_ = os.Remove(path)
		} else {
			_ = conn.Close()
		}
	}

	// 新的监听任务
	listener, err := net.Listen("unix", path)
	if err != nil {
		return err
	}
	events.On(events.EventQuit, func() {
		remotelogs.Println("NODE", "quit unix sock")
		_ = listener.Close()
	})

	go func() {
		for {
			_, err := listener.Accept()
			if err != nil {
				return
			}
		}
	}()

	return nil
}
