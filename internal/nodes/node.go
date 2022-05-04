package nodes

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/TeaOSLab/EdgeCommon/pkg/nodeconfigs"
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/caches"
	"github.com/TeaOSLab/EdgeNode/internal/configs"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/firewalls"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/TeaOSLab/EdgeNode/internal/iplibrary"
	"github.com/TeaOSLab/EdgeNode/internal/metrics"
	"github.com/TeaOSLab/EdgeNode/internal/ratelimit"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/rpc"
	"github.com/TeaOSLab/EdgeNode/internal/stats"
	"github.com/TeaOSLab/EdgeNode/internal/trackers"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/andybalholm/brotli"
	"github.com/iwind/TeaGo/Tea"
	"github.com/iwind/TeaGo/lists"
	"github.com/iwind/TeaGo/logs"
	"github.com/iwind/TeaGo/maps"
	"github.com/iwind/TeaGo/types"
	"github.com/iwind/gosock/pkg/gosock"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"runtime/debug"
	"sort"
	"sync"
	"syscall"
	"time"
)

var sharedNodeConfig *nodeconfigs.NodeConfig
var nodeTaskNotify = make(chan bool, 8)
var nodeConfigChangedNotify = make(chan bool, 8)
var nodeConfigUpdatedAt int64
var DaemonIsOn = false
var DaemonPid = 0

// Node 节点
type Node struct {
	isLoaded bool
	sock     *gosock.Sock
	locker   sync.Mutex

	maxCPU     int32
	maxThreads int
	timezone   string

	updatingServerMap map[int64]*serverconfigs.ServerConfig
}

func NewNode() *Node {
	return &Node{
		sock:              gosock.NewTmpSock(teaconst.ProcessName),
		maxThreads:        -1,
		maxCPU:            -1,
		updatingServerMap: map[int64]*serverconfigs.ServerConfig{},
	}
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

	// 处理异常
	this.handlePanic()

	// 监听signal
	this.listenSignals()

	// 启动事件
	events.Notify(events.EventStart)

	// 本地Sock
	err := this.listenSock()
	if err != nil {
		remotelogs.Error("NODE", err.Error())
		return
	}

	// 检查硬盘类型
	this.checkDisk()

	// 读取API配置
	err = this.syncConfig(0)
	if err != nil {
		_, err := nodeconfigs.SharedNodeConfig()
		if err != nil {
			// 无本地数据时，会尝试多次读取
			tryTimes := 0
			for {
				err := this.syncConfig(0)
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
		}
	}

	// 启动同步计时器
	this.startSyncTimer()

	// 状态变更计时器
	goman.New(func() {
		NewNodeStatusExecutor().Listen()
	})

	// 读取配置
	nodeConfig, err := nodeconfigs.SharedNodeConfig()
	if err != nil {
		remotelogs.Error("NODE", "start failed: read node config failed: "+err.Error())
		return
	}
	teaconst.NodeId = nodeConfig.Id
	teaconst.NodeIdString = types.String(teaconst.NodeId)
	err, serverErrors := nodeConfig.Init()
	if err != nil {
		remotelogs.Error("NODE", "init node config failed: "+err.Error())
		return
	}
	if len(serverErrors) > 0 {
		for _, serverErr := range serverErrors {
			remotelogs.ServerError(serverErr.Id, "NODE", serverErr.Message, nodeconfigs.NodeLogTypeServerConfigInitFailed, maps.Map{})
		}
	}
	sharedNodeConfig = nodeConfig
	this.onReload(nodeConfig)

	// 发送事件
	events.Notify(events.EventLoaded)

	// 设置rlimit
	_ = utils.SetRLimit(1024 * 1024)

	// 连接API
	goman.New(func() {
		NewAPIStream().Start()
	})

	// 统计
	goman.New(func() {
		stats.SharedTrafficStatManager.Start(func() *nodeconfigs.NodeConfig {
			return sharedNodeConfig
		})
	})
	goman.New(func() {
		stats.SharedHTTPRequestStatManager.Start()
	})

	// 启动端口
	err = sharedListenerManager.Start(nodeConfig)
	if err != nil {
		remotelogs.Error("NODE", "start failed: "+err.Error())
		return
	}

	// hold住进程
	select {}
}

// Daemon 实现守护进程
func (this *Node) Daemon() {
	isDebug := lists.ContainsString(os.Args, "debug")
	isDebug = true
	for {
		conn, err := this.sock.Dial()
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
				_ = os.Setenv("EdgeBackground", "on")

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

// 循环
func (this *Node) loop() error {
	var tr = trackers.Begin("CHECK_NODE_CONFIG_CHANGES")
	defer tr.End()

	// 检查api.yaml是否存在
	var apiConfigFile = Tea.ConfigFile("api.yaml")
	_, err := os.Stat(apiConfigFile)
	if err != nil {
		return nil
	}

	rpcClient, err := rpc.SharedRPC()
	if err != nil {
		return errors.New("create rpc client failed: " + err.Error())
	}

	var nodeCtx = rpcClient.Context()
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
			if task.ServerId > 0 {
				err = this.syncServerConfig(task.ServerId)
			} else {
				if !task.IsPrimary {
					// 我们等等主节点配置准备完毕
					time.Sleep(2 * time.Second)
				}
				err = this.syncConfig(task.Version)
			}
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
			goman.New(func() {
				sharedUpgradeManager.Start()
			})
		case "scriptsChanged":
			err = this.reloadCommonScripts()
			if err != nil {
				return errors.New("reload common scripts failed: " + err.Error())
			}

			// 修改为已同步
			_, err = rpcClient.NodeTaskRPC().ReportNodeTaskDone(nodeCtx, &pb.ReportNodeTaskDoneRequest{
				NodeTaskId: task.Id,
				IsOk:       true,
				Error:      "",
			})
			if err != nil {
				return err
			}
		case "nodeLevelChanged":
			levelInfoResp, err := rpcClient.NodeRPC().FindNodeLevelInfo(nodeCtx, &pb.FindNodeLevelInfoRequest{})
			if err != nil {
				return err
			}

			sharedNodeConfig.Level = levelInfoResp.Level

			var parentNodes = map[int64][]*nodeconfigs.ParentNodeConfig{}
			if len(levelInfoResp.ParentNodesMapJSON) > 0 {
				err = json.Unmarshal(levelInfoResp.ParentNodesMapJSON, &parentNodes)
				if err != nil {
					return errors.New("decode level info failed: " + err.Error())
				}
			}
			sharedNodeConfig.ParentNodes = parentNodes

			// 修改为已同步
			_, err = rpcClient.NodeTaskRPC().ReportNodeTaskDone(nodeCtx, &pb.ReportNodeTaskDoneRequest{
				NodeTaskId: task.Id,
				IsOk:       true,
				Error:      "",
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// 读取API配置
func (this *Node) syncConfig(taskVersion int64) error {
	this.locker.Lock()
	defer this.locker.Unlock()

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
				return errors.New("check cluster config failed: " + clusterErr.Error())
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
		Version:         -1, // 更新所有版本
		Compress:        true,
		NodeTaskVersion: taskVersion,
	})
	if err != nil {
		return errors.New("read config from rpc failed: " + err.Error())
	}
	if !configResp.IsChanged {
		return nil
	}

	configJSON := configResp.NodeJSON
	if configResp.IsCompressed {
		var reader = brotli.NewReader(bytes.NewReader(configJSON))
		var configBuf = &bytes.Buffer{}
		var buf = make([]byte, 32*1024)
		for {
			n, err := reader.Read(buf)
			if n > 0 {
				configBuf.Write(buf[:n])
			}
			if err != nil {
				break
			}
		}
		configJSON = configBuf.Bytes()
	}

	nodeConfigUpdatedAt = time.Now().Unix()

	nodeConfig := &nodeconfigs.NodeConfig{}
	err = json.Unmarshal(configJSON, nodeConfig)
	if err != nil {
		return errors.New("decode config failed: " + err.Error())
	}
	teaconst.NodeId = nodeConfig.Id
	teaconst.NodeIdString = types.String(teaconst.NodeId)

	// 写入到文件中
	err = nodeConfig.Save()
	if err != nil {
		return err
	}

	err, serverErrors := nodeConfig.Init()
	if err != nil {
		return err
	}
	if len(serverErrors) > 0 {
		for _, serverErr := range serverErrors {
			remotelogs.ServerError(serverErr.Id, "NODE", serverErr.Message, nodeconfigs.NodeLogTypeServerConfigInitFailed, maps.Map{})
		}
	}

	// 刷新配置
	if this.isLoaded {
		remotelogs.Println("NODE", "reloading config ...")
	} else {
		remotelogs.Println("NODE", "loading config ...")
	}

	this.onReload(nodeConfig)

	// 发送事件
	events.Notify(events.EventReload)

	if this.isLoaded {
		return sharedListenerManager.Start(nodeConfig)
	}

	this.isLoaded = true

	return nil
}

// 读取单个服务配置
func (this *Node) syncServerConfig(serverId int64) error {
	rpcClient, err := rpc.SharedRPC()
	if err != nil {
		return err
	}
	resp, err := rpcClient.ServerRPC().ComposeServerConfig(rpcClient.Context(), &pb.ComposeServerConfigRequest{ServerId: serverId})
	if err != nil {
		return err
	}

	this.locker.Lock()
	defer this.locker.Unlock()
	if len(resp.ServerConfigJSON) == 0 {
		this.updatingServerMap[serverId] = nil
	} else {
		var config = &serverconfigs.ServerConfig{}
		err = json.Unmarshal(resp.ServerConfigJSON, config)
		if err != nil {
			return err
		}
		this.updatingServerMap[serverId] = config
	}
	return nil
}

// 启动同步计时器
func (this *Node) startSyncTimer() {
	// TODO 这个时间间隔可以自行设置
	var taskTicker = time.NewTicker(60 * time.Second)
	var serverChangeTicker = time.NewTicker(5 * time.Second)

	events.OnKey(events.EventQuit, this, func() {
		remotelogs.Println("NODE", "quit sync timer")
		taskTicker.Stop()
		serverChangeTicker.Stop()
	})
	goman.New(func() {
		for {
			select {
			case <-taskTicker.C: // 定期执行
				err := this.loop()
				if err != nil {
					remotelogs.Error("NODE", "sync config error: "+err.Error())
					continue
				}
			case <-serverChangeTicker.C: // 服务变化
				this.reloadServer()
			case <-nodeTaskNotify: // 有新的更新任务
				err := this.loop()
				if err != nil {
					remotelogs.Error("NODE", "sync config error: "+err.Error())
					continue
				}
			case <-nodeConfigChangedNotify: // 节点变化通知
				err := this.syncConfig(0)
				if err != nil {
					remotelogs.Error("NODE", "sync config error: "+err.Error())
					continue
				}
			}
		}
	})
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

	logs.Println("[NODE]registering node to cluster ...")
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

// 监听一些信号
func (this *Node) listenSignals() {
	var queue = make(chan os.Signal, 8)
	signal.Notify(queue, syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL, syscall.SIGQUIT)
	goman.New(func() {
		for range queue {
			time.Sleep(100 * time.Millisecond)
			utils.Exit()
			return
		}
	})
}

// 监听本地sock
func (this *Node) listenSock() error {
	// 检查是否在运行
	if this.sock.IsListening() {
		reply, err := this.sock.Send(&gosock.Command{Code: "pid"})
		if err == nil {
			return errors.New("error: the process is already running, pid: " + maps.NewMap(reply.Params).GetString("pid"))
		} else {
			return errors.New("error: the process is already running")
		}
	}

	// 启动监听
	goman.New(func() {
		this.sock.OnCommand(func(cmd *gosock.Command) {
			switch cmd.Code {
			case "pid":
				_ = cmd.Reply(&gosock.Command{
					Code: "pid",
					Params: map[string]interface{}{
						"pid": os.Getpid(),
					},
				})
			case "info":
				exePath, _ := os.Executable()
				_ = cmd.Reply(&gosock.Command{
					Code: "info",
					Params: map[string]interface{}{
						"pid":     os.Getpid(),
						"version": teaconst.Version,
						"path":    exePath,
					},
				})
			case "stop":
				_ = cmd.ReplyOk()

				// 退出主进程
				events.Notify(events.EventQuit)
				utils.Exit()
			case "quit":
				_ = cmd.ReplyOk()
				_ = this.sock.Close()

				events.Notify(events.EventQuit)

				// 监控连接数，如果连接数为0，则退出进程
				goman.New(func() {
					for {
						countActiveConnections := sharedListenerManager.TotalActiveConnections()
						if countActiveConnections <= 0 {
							utils.Exit()
							return
						}
						time.Sleep(1 * time.Second)
					}
				})
			case "trackers":
				_ = cmd.Reply(&gosock.Command{
					Params: map[string]interface{}{
						"labels": trackers.SharedManager.Labels(),
					},
				})
			case "goman":
				var posMap = map[string]maps.Map{} // file#line => Map
				for _, instance := range goman.List() {
					var pos = instance.File + "#" + types.String(instance.Line)
					m, ok := posMap[pos]
					if ok {
						m["count"] = m["count"].(int) + 1
					} else {
						m = maps.Map{
							"pos":   pos,
							"count": 1,
						}
						posMap[pos] = m
					}
				}

				var result = []maps.Map{}
				for _, m := range posMap {
					result = append(result, m)
				}

				sort.Slice(result, func(i, j int) bool {
					return result[i]["count"].(int) > result[j]["count"].(int)
				})

				_ = cmd.Reply(&gosock.Command{
					Params: map[string]interface{}{
						"total":  runtime.NumGoroutine(),
						"result": result,
					},
				})
			case "conns":
				ipConns, serverConns := sharedClientConnLimiter.Conns()

				_ = cmd.Reply(&gosock.Command{
					Params: map[string]interface{}{
						"ipConns":     ipConns,
						"serverConns": serverConns,
						"total":       sharedListenerManager.TotalActiveConnections(),
						"limiter":     sharedConnectionsLimiter.Len(),
					},
				})
			case "dropIP":
				var m = maps.NewMap(cmd.Params)
				var ip = m.GetString("ip")
				var timeSeconds = m.GetInt("timeoutSeconds")
				err := firewalls.Firewall().DropSourceIP(ip, timeSeconds)
				if err != nil {
					_ = cmd.Reply(&gosock.Command{
						Params: map[string]interface{}{
							"error": err.Error(),
						},
					})
				} else {
					_ = cmd.ReplyOk()
				}
			case "rejectIP":
				var m = maps.NewMap(cmd.Params)
				var ip = m.GetString("ip")
				var timeSeconds = m.GetInt("timeoutSeconds")
				err := firewalls.Firewall().RejectSourceIP(ip, timeSeconds)
				if err != nil {
					_ = cmd.Reply(&gosock.Command{
						Params: map[string]interface{}{
							"error": err.Error(),
						},
					})
				} else {
					_ = cmd.ReplyOk()
				}
			case "removeIP":
				var m = maps.NewMap(cmd.Params)
				var ip = m.GetString("ip")
				err := firewalls.Firewall().RemoveSourceIP(ip)
				if err != nil {
					_ = cmd.Reply(&gosock.Command{
						Params: map[string]interface{}{
							"error": err.Error(),
						},
					})
				} else {
					_ = cmd.ReplyOk()
				}
			case "gc":
				runtime.GC()
				debug.FreeOSMemory()
				_ = cmd.ReplyOk()
			case "reload":
				err := this.syncConfig(0)
				if err != nil {
					_ = cmd.Reply(&gosock.Command{
						Params: map[string]interface{}{
							"error": err.Error(),
						},
					})
				} else {
					_ = cmd.ReplyOk()
				}
			}
		})

		err := this.sock.Listen()
		if err != nil {
			logs.Println("NODE", err.Error())
		}
	})

	events.OnKey(events.EventQuit, this, func() {
		logs.Println("NODE", "quit unix sock")
		_ = this.sock.Close()
	})

	return nil
}

// 重载配置调用
func (this *Node) onReload(config *nodeconfigs.NodeConfig) {
	nodeconfigs.ResetNodeConfig(config)
	sharedNodeConfig = config

	// 缓存策略
	caches.SharedManager.MaxDiskCapacity = config.MaxCacheDiskCapacity
	caches.SharedManager.MaxMemoryCapacity = config.MaxCacheMemoryCapacity
	caches.SharedManager.DiskDir = config.CacheDiskDir
	if len(config.HTTPCachePolicies) > 0 {
		caches.SharedManager.UpdatePolicies(config.HTTPCachePolicies)
	} else {
		caches.SharedManager.UpdatePolicies([]*serverconfigs.HTTPCachePolicy{})
	}

	// WAF策略
	sharedWAFManager.UpdatePolicies(config.FindAllFirewallPolicies())
	iplibrary.SharedActionManager.UpdateActions(config.FirewallActions)

	// 统计指标
	metrics.SharedManager.Update(config.MetricItems)

	// max cpu
	if config.MaxCPU != this.maxCPU {
		if config.MaxCPU > 0 && config.MaxCPU < int32(runtime.NumCPU()) {
			runtime.GOMAXPROCS(int(config.MaxCPU))
			remotelogs.Println("NODE", "[CPU]set max cpu to '"+types.String(config.MaxCPU)+"'")
		} else {
			var threads = runtime.NumCPU() * 4
			runtime.GOMAXPROCS(threads)
			remotelogs.Println("NODE", "[CPU]set max cpu to '"+types.String(threads)+"'")
		}

		this.maxCPU = config.MaxCPU
	}

	// max threads
	if config.MaxThreads != this.maxThreads {
		if config.MaxThreads > 0 {
			debug.SetMaxThreads(config.MaxThreads)
			remotelogs.Println("NODE", "[THREADS]set max threads to '"+types.String(config.MaxThreads)+"'")
		} else {
			debug.SetMaxThreads(nodeconfigs.DefaultMaxThreads)
			remotelogs.Println("NODE", "[THREADS]set max threads to '"+types.String(nodeconfigs.DefaultMaxThreads)+"'")
		}
		this.maxThreads = config.MaxThreads
	}

	// max tcp connections
	if config.TCPMaxConnections <= 0 {
		config.TCPMaxConnections = nodeconfigs.DefaultTCPMaxConnections
	}
	if config.TCPMaxConnections != sharedConnectionsLimiter.Count() {
		remotelogs.Println("NODE", "[TCP]changed tcp max connections to '"+types.String(config.TCPMaxConnections)+"'")

		sharedConnectionsLimiter.Close()
		sharedConnectionsLimiter = ratelimit.NewCounter(config.TCPMaxConnections)
	}

	// timezone
	var timeZone = config.TimeZone
	if len(timeZone) == 0 {
		timeZone = "Asia/Shanghai"
	}

	if this.timezone != timeZone {
		location, err := time.LoadLocation(timeZone)
		if err != nil {
			remotelogs.Error("NODE", "[TIMEZONE]change time zone failed: "+err.Error())
			return
		}

		remotelogs.Println("NODE", "[TIMEZONE]change time zone to '"+timeZone+"'")
		time.Local = location
		this.timezone = timeZone
	}

	// product information
	if config.ProductConfig != nil {
		teaconst.GlobalProductName = config.ProductConfig.Name
	}

	// DNS resolver
	if config.DNSResolver != nil {
		var err error
		switch config.DNSResolver.Type {
		case nodeconfigs.DNSResolverTypeGoNative:
			err = os.Setenv("GODEBUG", "netdns=go+2")
		case nodeconfigs.DNSResolverTypeCGO:
			err = os.Setenv("GODEBUG", "netdns=cgo+2")
		default:
			err = os.Unsetenv("GODEBUG")
		}
		if err != nil {
			remotelogs.Error("NODE", "[DNS_RESOLVER]set env failed: "+err.Error())
		}
	}
}

// reload server config
func (this *Node) reloadServer() {
	this.locker.Lock()
	defer this.locker.Unlock()

	if len(this.updatingServerMap) > 0 {
		var updatingServerMap = this.updatingServerMap
		this.updatingServerMap = map[int64]*serverconfigs.ServerConfig{}
		newNodeConfig, err := nodeconfigs.CloneNodeConfig(sharedNodeConfig)
		if err != nil {
			remotelogs.Error("NODE", "apply server config error: "+err.Error())
			return
		}
		for serverId, serverConfig := range updatingServerMap {
			if serverConfig != nil {
				newNodeConfig.AddServer(serverConfig)
			} else {
				newNodeConfig.RemoveServer(serverId)
			}
		}

		err, serverErrors := newNodeConfig.Init()
		if err != nil {
			remotelogs.Error("NODE", "apply server config error: "+err.Error())
			return
		}
		if len(serverErrors) > 0 {
			for _, serverErr := range serverErrors {
				remotelogs.ServerError(serverErr.Id, "NODE", serverErr.Message, nodeconfigs.NodeLogTypeServerConfigInitFailed, maps.Map{})
			}
		}

		this.onReload(newNodeConfig)

		err = sharedListenerManager.Start(newNodeConfig)
		if err != nil {
			remotelogs.Error("NODE", "apply server config error: "+err.Error())
		}
	}
}

func (this *Node) checkDisk() {
	if runtime.GOOS == "linux" {
		for _, path := range []string{
			"/sys/block/vda/queue/rotational",
			"/sys/block/sda/queue/rotational",
		} {
			data, err := ioutil.ReadFile(path)
			if err != nil {
				continue
			}
			if string(data) == "0" {
				teaconst.DiskIsFast = true
			}
			break
		}
	}
}
