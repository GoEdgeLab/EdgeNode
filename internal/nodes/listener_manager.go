package nodes

import (
	"fmt"
	"github.com/TeaOSLab/EdgeCommon/pkg/nodeconfigs"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/firewalls"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	executils "github.com/TeaOSLab/EdgeNode/internal/utils/exec"
	"github.com/iwind/TeaGo/Tea"
	"github.com/iwind/TeaGo/lists"
	"github.com/iwind/TeaGo/maps"
	"github.com/iwind/TeaGo/types"
	"net/url"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

var sharedListenerManager *ListenerManager

func init() {
	if !teaconst.IsMain {
		return
	}

	sharedListenerManager = NewListenerManager()
}

// ListenerManager 端口监听管理器
type ListenerManager struct {
	listenersMap  map[string]*Listener // addr => *Listener
	http3Listener *HTTPListener

	locker     sync.Mutex
	lastConfig *nodeconfigs.NodeConfig

	retryListenerMap map[string]*Listener // 需要重试的监听器 addr => Listener
	ticker           *time.Ticker

	firewalld         *firewalls.Firewalld
	lastPortStrings   string
	lastTCPPortRanges [][2]int
	lastUDPPortRanges [][2]int
}

// NewListenerManager 获取新对象
func NewListenerManager() *ListenerManager {
	var manager = &ListenerManager{
		listenersMap:     map[string]*Listener{},
		retryListenerMap: map[string]*Listener{},
		ticker:           time.NewTicker(1 * time.Minute),
		firewalld:        firewalls.NewFirewalld(),
	}

	// 提升测试效率
	if Tea.IsTesting() {
		manager.ticker = time.NewTicker(5 * time.Second)
	}

	goman.New(func() {
		for range manager.ticker.C {
			manager.retryListeners()
		}
	})

	return manager
}

// Start 启动监听
func (this *ListenerManager) Start(nodeConfig *nodeconfigs.NodeConfig) error {
	this.locker.Lock()
	defer this.locker.Unlock()

	// 重置数据
	this.retryListenerMap = map[string]*Listener{}

	// 检查是否有变化
	/**if this.lastConfig != nil && this.lastConfig.Version == node.Version {
		return nil
	}**/
	this.lastConfig = nodeConfig

	// 所有的新地址
	var groupAddrs = []string{}
	var availableServerGroups = nodeConfig.AvailableGroups()
	if !nodeConfig.IsOn {
		availableServerGroups = []*serverconfigs.ServerAddressGroup{}
	}

	if len(availableServerGroups) == 0 {
		remotelogs.Println("LISTENER_MANAGER", "no available servers to startup")
	}

	for _, group := range availableServerGroups {
		var addr = group.FullAddr()
		groupAddrs = append(groupAddrs, addr)
	}

	// 停掉老的
	for listenerKey, listener := range this.listenersMap {
		var addr = listener.FullAddr()
		if !lists.ContainsString(groupAddrs, addr) {
			remotelogs.Println("LISTENER_MANAGER", "close '"+addr+"'")
			_ = listener.Close()

			delete(this.listenersMap, listenerKey)
		}
	}

	// 启动新的或修改老的
	for _, group := range availableServerGroups {
		var addr = group.FullAddr()
		listener, ok := this.listenersMap[addr]
		if ok {
			// 不需要打印reload信息，防止日志数量过多
			listener.Reload(group)
		} else {
			remotelogs.Println("LISTENER_MANAGER", "listen '"+this.prettyAddress(addr)+"'")
			listener = NewListener()
			listener.Reload(group)
			err := listener.Listen()
			if err != nil {
				// 放入到重试队列中
				this.retryListenerMap[addr] = listener

				var firstServer = group.FirstServer()
				if firstServer == nil {
					remotelogs.Error("LISTENER_MANAGER", err.Error())
				} else {
					// 当前占用的进程名
					if strings.Contains(err.Error(), "in use") {
						portIndex := strings.LastIndex(addr, ":")
						if portIndex > 0 {
							var port = addr[portIndex+1:]
							var processName = this.findProcessNameWithPort(group.IsUDP(), port)
							if len(processName) > 0 {
								err = fmt.Errorf("%w (the process using port: '%s')", err, processName)
							}
						}
					}

					remotelogs.ServerError(firstServer.Id, "LISTENER_MANAGER", "listen '"+addr+"' failed: "+err.Error(), nodeconfigs.NodeLogTypeListenAddressFailed, maps.Map{"address": addr})
				}

				continue
			} else {
				// TODO 是否是从错误中恢复
			}
			this.listenersMap[addr] = listener
		}
	}

	// 加入到firewalld
	go this.addToFirewalld(groupAddrs)

	return nil
}

// TotalActiveConnections 获取总的活跃连接数
func (this *ListenerManager) TotalActiveConnections() int {
	this.locker.Lock()
	defer this.locker.Unlock()

	var total = 0
	for _, listener := range this.listenersMap {
		total += listener.listener.CountActiveConnections()
	}

	if this.http3Listener != nil {
		total += this.http3Listener.CountActiveConnections()
	}

	return total
}

// 返回更加友好格式的地址
func (this *ListenerManager) prettyAddress(addr string) string {
	u, err := url.Parse(addr)
	if err != nil {
		return addr
	}
	if regexp.MustCompile(`^:\d+$`).MatchString(u.Host) {
		u.Host = "*" + u.Host
	}
	return u.String()
}

// 重试失败的Listener
func (this *ListenerManager) retryListeners() {
	this.locker.Lock()
	defer this.locker.Unlock()

	for addr, listener := range this.retryListenerMap {
		err := listener.Listen()
		if err == nil {
			delete(this.retryListenerMap, addr)
			this.listenersMap[addr] = listener
			remotelogs.ServerSuccess(listener.group.FirstServer().Id, "LISTENER_MANAGER", "retry to listen '"+addr+"' successfully", nodeconfigs.NodeLogTypeListenAddressFailed, maps.Map{"address": addr})
		}
	}
}

func (this *ListenerManager) findProcessNameWithPort(isUdp bool, port string) string {
	if runtime.GOOS != "linux" {
		return ""
	}

	path, err := executils.LookPath("ss")
	if err != nil {
		return ""
	}

	var option = "t"
	if isUdp {
		option = "u"
	}

	var cmd = executils.NewTimeoutCmd(10*time.Second, path, "-"+option+"lpn", "sport = :"+port)
	cmd.WithStdout()
	err = cmd.Run()
	if err != nil {
		return ""
	}

	var matches = regexp.MustCompile(`(?U)\(\("(.+)",pid=\d+,fd=\d+\)\)`).FindStringSubmatch(cmd.Stdout())
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func (this *ListenerManager) addToFirewalld(groupAddrs []string) {
	if !sharedNodeConfig.AutoOpenPorts {
		return
	}

	if this.firewalld == nil || !this.firewalld.IsReady() {
		return
	}

	// HTTP/3相关端口
	var http3Ports = sharedNodeConfig.FindHTTP3Ports()
	if len(http3Ports) > 0 {
		for _, port := range http3Ports {
			var groupAddr = "udp://:" + types.String(port)
			if !lists.ContainsString(groupAddrs, groupAddr) {
				groupAddrs = append(groupAddrs, groupAddr)
			}
		}
	}

	// 组合端口号
	var portStrings = []string{}
	var udpPorts = []int{}
	var tcpPorts = []int{}
	for _, addr := range groupAddrs {
		var protocol = "tcp"
		if strings.HasPrefix(addr, "udp") {
			protocol = "udp"
		}

		var lastIndex = strings.LastIndex(addr, ":")
		if lastIndex > 0 {
			var portString = addr[lastIndex+1:]
			portStrings = append(portStrings, portString+"/"+protocol)

			switch protocol {
			case "tcp":
				tcpPorts = append(tcpPorts, types.Int(portString))
			case "udp":
				udpPorts = append(udpPorts, types.Int(portString))
			}
		}
	}
	if len(portStrings) == 0 {
		return
	}

	// 检查是否有变化
	sort.Strings(portStrings)
	var newPortStrings = strings.Join(portStrings, ",")
	if newPortStrings == this.lastPortStrings {
		return
	}
	this.locker.Lock()
	this.lastPortStrings = newPortStrings
	this.locker.Unlock()

	remotelogs.Println("FIREWALLD", "opening ports automatically ...")
	defer func() {
		remotelogs.Println("FIREWALLD", "open ports successfully")
	}()

	// 合并端口
	var tcpPortRanges = utils.MergePorts(tcpPorts)
	var udpPortRanges = utils.MergePorts(udpPorts)

	defer func() {
		this.locker.Lock()
		this.lastTCPPortRanges = tcpPortRanges
		this.lastUDPPortRanges = udpPortRanges
		this.locker.Unlock()
	}()

	// 删除老的不存在的端口
	var tcpPortRangesMap = map[string]bool{}
	var udpPortRangesMap = map[string]bool{}
	for _, portRange := range tcpPortRanges {
		tcpPortRangesMap[this.firewalld.PortRangeString(portRange, "tcp")] = true
	}
	for _, portRange := range udpPortRanges {
		udpPortRangesMap[this.firewalld.PortRangeString(portRange, "udp")] = true
	}

	for _, portRange := range this.lastTCPPortRanges {
		var s = this.firewalld.PortRangeString(portRange, "tcp")
		_, ok := tcpPortRangesMap[s]
		if ok {
			continue
		}
		remotelogs.Println("FIREWALLD", "remove port '"+s+"'")
		_ = this.firewalld.RemovePortRangePermanently(portRange, "tcp")
	}
	for _, portRange := range this.lastUDPPortRanges {
		var s = this.firewalld.PortRangeString(portRange, "udp")
		_, ok := udpPortRangesMap[s]
		if ok {
			continue
		}
		remotelogs.Println("FIREWALLD", "remove port '"+s+"'")
		_ = this.firewalld.RemovePortRangePermanently(portRange, "udp")
	}

	// 添加新的
	_ = this.firewalld.AllowPortRangesPermanently(tcpPortRanges, "tcp")
	_ = this.firewalld.AllowPortRangesPermanently(udpPortRanges, "udp")
}

func (this *ListenerManager) reloadFirewalld() {
	this.locker.Lock()
	defer this.locker.Unlock()

	var nodeConfig = sharedNodeConfig

	// 所有的新地址
	var groupAddrs = []string{}
	var availableServerGroups = nodeConfig.AvailableGroups()
	if !nodeConfig.IsOn {
		availableServerGroups = []*serverconfigs.ServerAddressGroup{}
	}

	if len(availableServerGroups) == 0 {
		remotelogs.Println("LISTENER_MANAGER", "no available servers to startup")
	}

	for _, group := range availableServerGroups {
		var addr = group.FullAddr()
		groupAddrs = append(groupAddrs, addr)
	}

	go this.addToFirewalld(groupAddrs)
}
