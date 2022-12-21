package nodes

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/TeaOSLab/EdgeCommon/pkg/configutils"
	iplib "github.com/TeaOSLab/EdgeCommon/pkg/iplibrary"
	"github.com/TeaOSLab/EdgeCommon/pkg/nodeconfigs"
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/ddosconfigs"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/firewallconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/caches"
	"github.com/TeaOSLab/EdgeNode/internal/configs"
	"github.com/TeaOSLab/EdgeNode/internal/conns"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/firewalls"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/TeaOSLab/EdgeNode/internal/iplibrary"
	"github.com/TeaOSLab/EdgeNode/internal/metrics"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/rpc"
	"github.com/TeaOSLab/EdgeNode/internal/stats"
	"github.com/TeaOSLab/EdgeNode/internal/trackers"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	_ "github.com/TeaOSLab/EdgeNode/internal/utils/clock" // 触发时钟更新
	"github.com/TeaOSLab/EdgeNode/internal/utils/jsonutils"
	"github.com/TeaOSLab/EdgeNode/internal/waf"
	"github.com/andybalholm/brotli"
	"github.com/iwind/TeaGo/Tea"
	"github.com/iwind/TeaGo/lists"
	"github.com/iwind/TeaGo/maps"
	"github.com/iwind/TeaGo/types"
	"github.com/iwind/gosock/pkg/gosock"
	"gopkg.in/yaml.v3"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
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

	oldMaxCPU               int32
	oldMaxThreads           int
	oldTimezone             string
	oldHTTPCachePolicies    []*serverconfigs.HTTPCachePolicy
	oldHTTPFirewallPolicies []*firewallconfigs.HTTPFirewallPolicy
	oldFirewallActions      []*firewallconfigs.FirewallActionConfig
	oldMetricItems          []*serverconfigs.MetricItemConfig

	updatingServerMap map[int64]*serverconfigs.ServerConfig

	lastAPINodeVersion int64
	lastAPINodeAddrs   []string // 以前的API节点地址

	lastTaskVersion int64
}

func NewNode() *Node {
	return &Node{
		sock:              gosock.NewTmpSock(teaconst.ProcessName),
		oldMaxThreads:     -1,
		oldMaxCPU:         -1,
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
	_, err = rpcClient.APINodeRPC.FindCurrentAPINodeVersion(rpcClient.Context(), &pb.FindCurrentAPINodeVersionRequest{})
	if err != nil {
		return errors.New("test rpc failed: " + err.Error())
	}

	return nil
}

// Start 启动
func (this *Node) Start() {
	// 设置netdns
	// 这个需要放在所有网络访问的最前面
	_ = os.Setenv("GODEBUG", "netdns=go")

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

	// 本地Sock
	err := this.listenSock()
	if err != nil {
		remotelogs.Error("NODE", err.Error())
		return
	}

	// 启动IP库
	remotelogs.Println("NODE", "initializing ip library ...")
	err = iplib.InitDefault()
	if err != nil {
		remotelogs.Error("NODE", "initialize ip library failed: "+err.Error())
	}

	// 检查硬盘类型
	this.checkDisk()

	// 启动事件
	events.Notify(events.EventStart)

	// 读取API配置
	remotelogs.Println("NODE", "init config ...")
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

	// 更新IP库
	goman.New(func() {
		iplib.NewUpdater(NewIPLibraryUpdater(), 10*time.Minute).Start()
	})

	// 监控节点运行状态
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
	this.onReload(nodeConfig, true)

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
		stats.SharedTrafficStatManager.Start()
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
	teaconst.IsDaemon = true

	var isDebug = lists.ContainsString(os.Args, "debug")
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

				var cmd = exec.Command(exe)
				var buf = &bytes.Buffer{}
				cmd.Stderr = buf
				err = cmd.Start()
				if err != nil {
					return err
				}
				err = cmd.Wait()
				if err != nil {
					if isDebug {
						log.Println("[DAEMON]" + buf.String())
					}
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

	tasksResp, err := rpcClient.NodeTaskRPC.FindNodeTasks(rpcClient.Context(), &pb.FindNodeTasksRequest{
		Version: this.lastTaskVersion,
	})
	if err != nil {
		if rpc.IsConnError(err) && !Tea.IsTesting() {
			return nil
		}
		return errors.New("read node tasks failed: " + err.Error())
	}
	for _, task := range tasksResp.NodeTasks {
		err := this.execTask(rpcClient, task)
		if !this.finishTask(task.Id, task.Version, err) {
			// 防止失败的任务无法重试
			break
		}
	}

	return nil
}

// 执行任务
func (this *Node) execTask(rpcClient *rpc.RPCClient, task *pb.NodeTask) error {
	switch task.Type {
	case "ipItemChanged":
		// 防止阻塞
		select {
		case iplibrary.IPListUpdateNotify <- true:
		default:

		}
	case "configChanged":
		if task.ServerId > 0 {
			return this.syncServerConfig(task.ServerId)
		}
		if !task.IsPrimary {
			// 我们等等主节点配置准备完毕
			time.Sleep(2 * time.Second)
		}
		return this.syncConfig(task.Version)
	case "nodeVersionChanged":
		if !sharedUpgradeManager.IsInstalling() {
			goman.New(func() {
				sharedUpgradeManager.Start()
			})
		}
	case "scriptsChanged":
		err := this.reloadCommonScripts()
		if err != nil {
			return errors.New("reload common scripts failed: " + err.Error())
		}
	case "nodeLevelChanged":
		levelInfoResp, err := rpcClient.NodeRPC.FindNodeLevelInfo(rpcClient.Context(), &pb.FindNodeLevelInfoRequest{})
		if err != nil {
			return err
		}

		if sharedNodeConfig != nil {
			sharedNodeConfig.Level = levelInfoResp.Level
		}

		var parentNodes = map[int64][]*nodeconfigs.ParentNodeConfig{}
		if len(levelInfoResp.ParentNodesMapJSON) > 0 {
			err = json.Unmarshal(levelInfoResp.ParentNodesMapJSON, &parentNodes)
			if err != nil {
				return errors.New("decode level info failed: " + err.Error())
			}
		}

		if sharedNodeConfig != nil {
			sharedNodeConfig.ParentNodes = parentNodes
		}
	case "ddosProtectionChanged":
		resp, err := rpcClient.NodeRPC.FindNodeDDoSProtection(rpcClient.Context(), &pb.FindNodeDDoSProtectionRequest{})
		if err != nil {
			return err
		}
		if len(resp.DdosProtectionJSON) == 0 {
			if sharedNodeConfig != nil {
				sharedNodeConfig.DDoSProtection = nil
			}
			return nil
		}

		var ddosProtectionConfig = &ddosconfigs.ProtectionConfig{}
		err = json.Unmarshal(resp.DdosProtectionJSON, ddosProtectionConfig)
		if err != nil {
			return errors.New("decode DDoS protection config failed: " + err.Error())
		}

		if ddosProtectionConfig != nil && sharedNodeConfig != nil {
			sharedNodeConfig.DDoSProtection = ddosProtectionConfig
		}

		err = firewalls.SharedDDoSProtectionManager.Apply(ddosProtectionConfig)
		if err != nil {
			// 不阻塞
			remotelogs.Warn("NODE", "apply DDoS protection failed: "+err.Error())
			return nil
		}
	case "globalServerConfigChanged":
		resp, err := rpcClient.NodeRPC.FindNodeGlobalServerConfig(rpcClient.Context(), &pb.FindNodeGlobalServerConfigRequest{})
		if err != nil {
			return err
		}
		if len(resp.GlobalServerConfigJSON) > 0 {
			var globalServerConfig = serverconfigs.DefaultGlobalServerConfig()
			err = json.Unmarshal(resp.GlobalServerConfigJSON, globalServerConfig)
			if err != nil {
				return errors.New("decode global server config failed: " + err.Error())
			}

			if globalServerConfig != nil {
				err = globalServerConfig.Init()
				if err != nil {
					return errors.New("validate global server config failed: " + err.Error())
				}
				if sharedNodeConfig != nil {
					sharedNodeConfig.GlobalServerConfig = globalServerConfig
				}
			}
		}
	case "userServersStateChanged":
		if task.UserId > 0 {
			resp, err := rpcClient.UserRPC.CheckUserServersState(rpcClient.Context(), &pb.CheckUserServersStateRequest{UserId: task.UserId})
			if err != nil {
				return err
			}

			SharedUserManager.UpdateUserServersIsEnabled(task.UserId, resp.IsEnabled)

			if resp.IsEnabled {
				err = this.syncUserServersConfig(task.UserId)
				if err != nil {
					return err
				}
			}
		}
	default:
		remotelogs.Error("NODE", "task '"+types.String(task.Id)+"', type '"+task.Type+"' has not been handled")
	}

	return nil
}

// 标记任务完成
func (this *Node) finishTask(taskId int64, taskVersion int64, taskErr error) (success bool) {
	if taskId <= 0 {
		return true
	}

	rpcClient, err := rpc.SharedRPC()
	if err != nil {
		remotelogs.Debug("NODE", "create rpc client failed: "+err.Error())
		return false
	}

	var isOk = taskErr == nil
	if isOk && taskVersion > this.lastTaskVersion {
		this.lastTaskVersion = taskVersion
	}

	var errMsg = ""
	if taskErr != nil {
		errMsg = taskErr.Error()
	}

	_, err = rpcClient.NodeTaskRPC.ReportNodeTaskDone(rpcClient.Context(), &pb.ReportNodeTaskDoneRequest{
		NodeTaskId: taskId,
		IsOk:       isOk,
		Error:      errMsg,
	})
	success = err == nil

	if err != nil {
		// 连接错误不需要上报到服务中心
		if rpc.IsConnError(err) {
			remotelogs.Debug("NODE", "report task done failed: "+err.Error())
		} else {
			remotelogs.Error("NODE", "report task done failed: "+err.Error())
		}
	}

	return success
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
					return errors.New("can not find config file 'configs/api.yaml'")
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
	// TODO 这里考虑只同步版本号有变更的
	configResp, err := rpcClient.NodeRPC.FindCurrentNodeConfig(rpcClient.Context(), &pb.FindCurrentNodeConfigRequest{
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

	var configJSON = configResp.NodeJSON
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

	var nodeConfig = &nodeconfigs.NodeConfig{}
	err = json.Unmarshal(configJSON, nodeConfig)
	if err != nil {
		return errors.New("decode config failed: " + err.Error())
	}
	teaconst.NodeId = nodeConfig.Id
	teaconst.NodeIdString = types.String(teaconst.NodeId)

	// 检查时间是否一致
	// 这个需要在 teaconst.NodeId 设置之后，因为上报到API节点的时候需要节点ID
	if configResp.Timestamp > 0 {
		var timestampDelta = configResp.Timestamp - time.Now().Unix()
		if timestampDelta > 60 || timestampDelta < -60 {
			remotelogs.Error("NODE", "node timestamp ('"+types.String(time.Now().Unix())+"') is not same as api node ('"+types.String(configResp.Timestamp)+"'), please sync the time")
		}
	}

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

	this.onReload(nodeConfig, true)

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
	resp, err := rpcClient.ServerRPC.ComposeServerConfig(rpcClient.Context(), &pb.ComposeServerConfigRequest{ServerId: serverId})
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

// 同步某个用户下的所有服务配置
func (this *Node) syncUserServersConfig(userId int64) error {
	rpcClient, err := rpc.SharedRPC()
	if err != nil {
		return err
	}
	serverConfigsResp, err := rpcClient.ServerRPC.ComposeAllUserServersConfig(rpcClient.Context(), &pb.ComposeAllUserServersConfigRequest{
		UserId: userId,
	})
	if err != nil {
		return err
	}
	if len(serverConfigsResp.ServersConfigJSON) == 0 {
		return nil
	}
	var serverConfigs = []*serverconfigs.ServerConfig{}
	err = json.Unmarshal(serverConfigsResp.ServersConfigJSON, &serverConfigs)
	if err != nil {
		return err
	}
	this.locker.Lock()
	defer this.locker.Unlock()

	for _, config := range serverConfigs {
		this.updatingServerMap[config.Id] = config
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
	data, err := os.ReadFile(configFile)
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

	remotelogs.Debug("NODE", "registering node to cluster ...")
	resp, err := rpcClient.NodeRPC.RegisterClusterNode(rpcClient.ClusterContext(config.ClusterId, config.Secret), &pb.RegisterClusterNodeRequest{Name: HOSTNAME})
	if err != nil {
		return err
	}
	remotelogs.Debug("NODE", "registered successfully")

	// 写入到配置文件中
	if len(resp.Endpoints) == 0 {
		resp.Endpoints = []string{}
	}
	var apiConfig = &configs.APIConfig{
		RPC: struct {
			Endpoints     []string `yaml:"endpoints" json:"endpoints"`
			DisableUpdate bool     `yaml:"disableUpdate" json:"disableUpdate"`
		}{
			Endpoints:     resp.Endpoints,
			DisableUpdate: false,
		},
		NodeId: resp.UniqueId,
		Secret: resp.Secret,
	}
	remotelogs.Debug("NODE", "writing 'configs/api.yaml' ...")
	err = apiConfig.WriteFile(Tea.ConfigFile("api.yaml"))
	if err != nil {
		return err
	}
	remotelogs.Debug("NODE", "wrote 'configs/api.yaml' successfully")

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
			return errors.New("error: the process is already running, pid: " + types.String(maps.NewMap(reply.Params).GetInt("pid")))
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
				time.Sleep(100 * time.Millisecond)
				utils.Exit()
			case "quit":
				_ = cmd.ReplyOk()
				_ = this.sock.Close()

				events.Notify(events.EventQuit)
				events.Notify(events.EventTerminated)

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
				var connMaps = []maps.Map{}
				var connMap = conns.SharedMap.AllConns()
				for _, conn := range connMap {
					var createdAt int64
					var lastReadAt int64
					var lastWriteAt int64
					var lastErrString = ""
					clientConn, ok := conn.(*ClientConn)
					if ok {
						createdAt = clientConn.CreatedAt()
						lastReadAt = clientConn.LastReadAt()
						lastWriteAt = clientConn.LastWriteAt()

						var lastErr = clientConn.LastErr()
						if lastErr != nil {
							lastErrString = lastErr.Error()
						}
					}
					var age int64
					var lastReadAge int64
					var lastWriteAge int64
					var currentTime = time.Now().Unix()
					if createdAt > 0 {
						age = currentTime - createdAt
					}
					if lastReadAt > 0 {
						lastReadAge = currentTime - lastReadAt
					}
					if lastWriteAt > 0 {
						lastWriteAge = currentTime - lastWriteAt
					}

					connMaps = append(connMaps, maps.Map{
						"addr":     conn.RemoteAddr().String(),
						"age":      age,
						"readAge":  lastReadAge,
						"writeAge": lastWriteAge,
						"lastErr":  lastErrString,
					})
				}
				sort.Slice(connMaps, func(i, j int) bool {
					var m1 = connMaps[i]
					var m2 = connMaps[j]
					return m1.GetInt64("age") < m2.GetInt64("age")
				})

				_ = cmd.Reply(&gosock.Command{
					Params: map[string]interface{}{
						"conns": connMaps,
						"total": len(connMaps),
					},
				})
			case "dropIP":
				var m = maps.NewMap(cmd.Params)
				var ip = m.GetString("ip")
				var timeSeconds = m.GetInt("timeoutSeconds")
				var async = m.GetBool("async")
				err := firewalls.Firewall().DropSourceIP(ip, timeSeconds, async)
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
			case "closeIP":
				var m = maps.NewMap(cmd.Params)
				var ip = m.GetString("ip")
				conns.SharedMap.CloseIPConns(ip)
				_ = cmd.ReplyOk()
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
			case "accesslog":
				err := sharedHTTPAccessLogViewer.Start()
				if err != nil {
					_ = cmd.Reply(&gosock.Command{
						Code: "error",
						Params: map[string]interface{}{
							"message": "start failed: " + err.Error(),
						},
					})
				} else {
					_ = cmd.ReplyOk()
				}
			case "bandwidth":
				var m = stats.SharedBandwidthStatManager.Map()
				_ = cmd.Reply(&gosock.Command{Params: maps.Map{
					"stats": m,
				}})
			}
		})

		err := this.sock.Listen()
		if err != nil {
			remotelogs.Debug("NODE", err.Error())
		}
	})

	events.OnKey(events.EventQuit, this, func() {
		remotelogs.Debug("NODE", "quit unix sock")
		_ = this.sock.Close()
	})

	return nil
}

// 重载配置调用
func (this *Node) onReload(config *nodeconfigs.NodeConfig, reloadAll bool) {
	nodeconfigs.ResetNodeConfig(config)
	sharedNodeConfig = config

	if reloadAll {
		// 缓存策略
		var subDirs = config.CacheDiskSubDirs
		for _, subDir := range subDirs {
			subDir.Path = filepath.Clean(subDir.Path)
		}
		if len(subDirs) > 0 {
			sort.Slice(subDirs, func(i, j int) bool {
				return subDirs[i].Path < subDirs[j].Path
			})
		}

		var cachePoliciesChanged = !jsonutils.Equal(caches.SharedManager.MaxDiskCapacity, config.MaxCacheDiskCapacity) ||
			!jsonutils.Equal(caches.SharedManager.MaxMemoryCapacity, config.MaxCacheMemoryCapacity) ||
			!jsonutils.Equal(caches.SharedManager.MainDiskDir, config.CacheDiskDir) ||
			!jsonutils.Equal(caches.SharedManager.SubDiskDirs, subDirs) ||
			!jsonutils.Equal(this.oldHTTPCachePolicies, config.HTTPCachePolicies)

		caches.SharedManager.MaxDiskCapacity = config.MaxCacheDiskCapacity
		caches.SharedManager.MaxMemoryCapacity = config.MaxCacheMemoryCapacity
		caches.SharedManager.MainDiskDir = config.CacheDiskDir
		caches.SharedManager.SubDiskDirs = subDirs

		if cachePoliciesChanged {
			// copy
			this.oldHTTPCachePolicies = []*serverconfigs.HTTPCachePolicy{}
			err := jsonutils.Copy(&this.oldHTTPCachePolicies, config.HTTPCachePolicies)
			if err != nil {
				remotelogs.Error("NODE", "onReload: copy HTTPCachePolicies failed: "+err.Error())
			}

			// update
			if len(config.HTTPCachePolicies) > 0 {
				caches.SharedManager.UpdatePolicies(config.HTTPCachePolicies)
			} else {
				caches.SharedManager.UpdatePolicies([]*serverconfigs.HTTPCachePolicy{})
			}
		}
	}

	// WAF策略
	// 包含了服务里的WAF策略，所以需要整体更新
	var allFirewallPolicies = config.FindAllFirewallPolicies()
	if !jsonutils.Equal(allFirewallPolicies, this.oldHTTPFirewallPolicies) {
		// copy
		this.oldHTTPFirewallPolicies = []*firewallconfigs.HTTPFirewallPolicy{}
		err := jsonutils.Copy(&this.oldHTTPFirewallPolicies, allFirewallPolicies)
		if err != nil {
			remotelogs.Error("NODE", "onReload: copy HTTPFirewallPolicies failed: "+err.Error())
		}

		// update
		waf.SharedWAFManager.UpdatePolicies(allFirewallPolicies)
	}

	if reloadAll {
		if !jsonutils.Equal(config.FirewallActions, this.oldFirewallActions) {
			// copy
			this.oldFirewallActions = []*firewallconfigs.FirewallActionConfig{}
			err := jsonutils.Copy(&this.oldFirewallActions, config.FirewallActions)
			if err != nil {
				remotelogs.Error("NODE", "onReload: copy FirewallActionConfigs failed: "+err.Error())
			}

			// update
			iplibrary.SharedActionManager.UpdateActions(config.FirewallActions)
		}

		// 统计指标
		if !jsonutils.Equal(this.oldMetricItems, config.MetricItems) {
			// copy
			this.oldMetricItems = []*serverconfigs.MetricItemConfig{}
			err := jsonutils.Copy(&this.oldMetricItems, config.MetricItems)
			if err != nil {
				remotelogs.Error("NODE", "onReload: copy MetricItemConfigs failed: "+err.Error())
			}

			// update
			metrics.SharedManager.Update(config.MetricItems)
		}

		// max cpu
		if config.MaxCPU != this.oldMaxCPU {
			if config.MaxCPU > 0 && config.MaxCPU < int32(runtime.NumCPU()) {
				runtime.GOMAXPROCS(int(config.MaxCPU))
				remotelogs.Println("NODE", "[CPU]set max cpu to '"+types.String(config.MaxCPU)+"'")
			} else {
				var threads = runtime.NumCPU() * 4
				runtime.GOMAXPROCS(threads)
				remotelogs.Println("NODE", "[CPU]set max cpu to '"+types.String(threads)+"'")
			}

			this.oldMaxCPU = config.MaxCPU
		}

		// max threads
		if config.MaxThreads != this.oldMaxThreads {
			if config.MaxThreads > 0 {
				debug.SetMaxThreads(config.MaxThreads)
				remotelogs.Println("NODE", "[THREADS]set max threads to '"+types.String(config.MaxThreads)+"'")
			} else {
				debug.SetMaxThreads(nodeconfigs.DefaultMaxThreads)
				remotelogs.Println("NODE", "[THREADS]set max threads to '"+types.String(nodeconfigs.DefaultMaxThreads)+"'")
			}
			this.oldMaxThreads = config.MaxThreads
		}

		// timezone
		var timeZone = config.TimeZone
		if len(timeZone) == 0 {
			timeZone = "Asia/Shanghai"
		}

		if this.oldTimezone != timeZone {
			location, err := time.LoadLocation(timeZone)
			if err != nil {
				remotelogs.Error("NODE", "[TIMEZONE]change time zone failed: "+err.Error())
				return
			}

			remotelogs.Println("NODE", "[TIMEZONE]change time zone to '"+timeZone+"'")
			time.Local = location
			this.oldTimezone = timeZone
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
				err = os.Setenv("GODEBUG", "netdns=go")
			case nodeconfigs.DNSResolverTypeCGO:
				err = os.Setenv("GODEBUG", "netdns=cgo")
			default:
				// 默认使用go原生
				err = os.Setenv("GODEBUG", "netdns=go")
			}
			if err != nil {
				remotelogs.Error("NODE", "[DNS_RESOLVER]set env failed: "+err.Error())
			}
		} else {
			// 默认使用go原生
			err := os.Setenv("GODEBUG", "netdns=go")
			if err != nil {
				remotelogs.Error("NODE", "[DNS_RESOLVER]set env failed: "+err.Error())
			}
		}

		// API Node地址，这里不限制是否为空，因为在为空时仍然要有对应的处理
		this.changeAPINodeAddrs(config.APINodeAddrs)
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

		this.onReload(newNodeConfig, false)

		err = sharedListenerManager.Start(newNodeConfig)
		if err != nil {
			remotelogs.Error("NODE", "apply server config error: "+err.Error())
		}
	}
}

// 检查硬盘
func (this *Node) checkDisk() {
	if runtime.GOOS != "linux" {
		return
	}
	for n := 'a'; n <= 'z'; n++ {
		for _, path := range []string{
			"/sys/block/vd" + string(n) + "/queue/rotational",
			"/sys/block/sd" + string(n) + "/queue/rotational",
		} {
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			if string(data) == "0" {
				teaconst.DiskIsFast = true
			}
			return
		}
	}
}

// 检查API节点地址
func (this *Node) changeAPINodeAddrs(apiNodeAddrs []*serverconfigs.NetworkAddressConfig) {
	var addrs = []string{}
	for _, addr := range apiNodeAddrs {
		err := addr.Init()
		if err != nil {
			remotelogs.Error("NODE", "changeAPINodeAddrs: validate api node address '"+configutils.QuoteIP(addr.Host)+":"+addr.PortRange+"' failed: "+err.Error())
		} else {
			addrs = append(addrs, addr.FullAddresses()...)
		}
	}
	sort.Strings(addrs)

	if utils.EqualStrings(this.lastAPINodeAddrs, addrs) {
		return
	}

	this.lastAPINodeAddrs = addrs

	config, err := configs.LoadAPIConfig()
	if err != nil {
		remotelogs.Error("NODE", "changeAPINodeAddrs: "+err.Error())
		return
	}
	if config == nil {
		return
	}
	var oldEndpoints = config.RPC.Endpoints

	rpcClient, err := rpc.SharedRPC()
	if err != nil {
		return
	}
	if len(addrs) > 0 {
		this.lastAPINodeVersion++
		var v = this.lastAPINodeVersion

		// 异步检测，防止阻塞
		go func(v int64) {
			// 测试新的API节点地址
			if rpcClient.TestEndpoints(addrs) {
				config.RPC.Endpoints = addrs
			} else {
				config.RPC.Endpoints = oldEndpoints
				this.lastAPINodeAddrs = nil // 恢复为空，以便于下次更新重试
			}

			// 检查测试中间有无新的变更
			if v != this.lastAPINodeVersion {
				return
			}

			err = rpcClient.UpdateConfig(config)
			if err != nil {
				remotelogs.Error("NODE", "changeAPINodeAddrs: update rpc config failed: "+err.Error())
			}
		}(v)
		return
	}

	err = rpcClient.UpdateConfig(config)
	if err != nil {
		remotelogs.Error("NODE", "changeAPINodeAddrs: update rpc config failed: "+err.Error())
	}
}
