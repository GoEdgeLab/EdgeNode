package nodes

import (
	"bytes"
	"errors"
	"github.com/TeaOSLab/EdgeCommon/pkg/nodeconfigs"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/iwind/TeaGo/Tea"
	"github.com/iwind/TeaGo/lists"
	"github.com/iwind/TeaGo/maps"
	"net/url"
	"os/exec"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

var sharedListenerManager = NewListenerManager()

// ListenerManager 端口监听管理器
type ListenerManager struct {
	listenersMap map[string]*Listener // addr => *Listener
	locker       sync.Mutex
	lastConfig   *nodeconfigs.NodeConfig

	retryListenerMap map[string]*Listener // 需要重试的监听器 addr => Listener
	ticker           *time.Ticker

	lastPortStrings string
}

// NewListenerManager 获取新对象
func NewListenerManager() *ListenerManager {
	manager := &ListenerManager{
		listenersMap:     map[string]*Listener{},
		retryListenerMap: map[string]*Listener{},
		ticker:           time.NewTicker(1 * time.Minute),
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
func (this *ListenerManager) Start(node *nodeconfigs.NodeConfig) error {
	this.locker.Lock()
	defer this.locker.Unlock()

	// 重置数据
	this.retryListenerMap = map[string]*Listener{}

	// 检查是否有变化
	/**if this.lastConfig != nil && this.lastConfig.Version == node.Version {
		return nil
	}**/
	this.lastConfig = node

	// 初始化
	err, _ := node.Init()
	if err != nil {
		return err
	}

	// 所有的新地址
	groupAddrs := []string{}
	availableServerGroups := node.AvailableGroups()
	if !node.IsOn {
		availableServerGroups = []*serverconfigs.ServerAddressGroup{}
	}

	if len(availableServerGroups) == 0 {
		remotelogs.Println("LISTENER_MANAGER", "no available servers to startup")
	}

	for _, group := range availableServerGroups {
		addr := group.FullAddr()
		groupAddrs = append(groupAddrs, addr)
	}

	// 停掉老的
	for listenerKey, listener := range this.listenersMap {
		addr := listener.FullAddr()
		if !lists.ContainsString(groupAddrs, addr) {
			remotelogs.Println("LISTENER_MANAGER", "close '"+addr+"'")
			_ = listener.Close()

			delete(this.listenersMap, listenerKey)
		}
	}

	// 启动新的或修改老的
	for _, group := range availableServerGroups {
		addr := group.FullAddr()
		listener, ok := this.listenersMap[addr]
		if ok {
			remotelogs.Println("LISTENER_MANAGER", "reload '"+this.prettyAddress(addr)+"'")
			listener.Reload(group)
		} else {
			remotelogs.Println("LISTENER_MANAGER", "listen '"+this.prettyAddress(addr)+"'")
			listener = NewListener()
			listener.Reload(group)
			err := listener.Listen()
			if err != nil {
				// 放入到重试队列中
				this.retryListenerMap[addr] = listener

				firstServer := group.FirstServer()
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
								err = errors.New(err.Error() + " (the process using port: '" + processName + "')")
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
	this.addToFirewalld(groupAddrs)

	return nil
}

// TotalActiveConnections 获取总的活跃连接数
func (this *ListenerManager) TotalActiveConnections() int {
	this.locker.Lock()
	defer this.locker.Unlock()

	total := 0
	for _, listener := range this.listenersMap {
		total += listener.listener.CountActiveListeners()
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

	path, err := exec.LookPath("ss")
	if err != nil {
		return ""
	}

	var option = "t"
	if isUdp {
		option = "u"
	}

	var cmd = exec.Command(path, "-"+option+"lpn", "sport = :"+port)
	var output = &bytes.Buffer{}
	cmd.Stdout = output
	err = cmd.Run()
	if err != nil {
		return ""
	}

	var matches = regexp.MustCompile(`(?U)\(\("(.+)",pid=\d+,fd=\d+\)\)`).FindStringSubmatch(output.String())
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func (this *ListenerManager) addToFirewalld(groupAddrs []string) {
	if !sharedNodeConfig.AutoOpenPorts {
		return
	}

	// 组合端口号
	var ports = []string{}
	for _, addr := range groupAddrs {
		var protocol = "tcp"
		if strings.HasPrefix(addr, "udp") {
			protocol = "udp"
		}

		var lastIndex = strings.LastIndex(addr, ":")
		if lastIndex > 0 {
			var portString = addr[lastIndex+1:]
			ports = append(ports, portString+"/"+protocol)
		}
	}
	if len(ports) == 0 {
		return
	}

	// 检查是否有变化
	sort.Strings(ports)
	var newPortStrings = strings.Join(ports, ",")
	if newPortStrings == this.lastPortStrings {
		return
	}
	this.lastPortStrings = newPortStrings

	firewallCmd, err := exec.LookPath("firewall-cmd")
	if err != nil || len(firewallCmd) == 0 {
		return
	}

	remotelogs.Println("FIREWALLD", "open ports automatically")
	for _, port := range ports {
		{
			// TODO 需要支持sudo
			var cmd = exec.Command(firewallCmd, "--add-port="+port, "--permanent")
			err = cmd.Run()
			if err != nil {
				remotelogs.Warn("FIREWALLD", "'"+cmd.String()+"': "+err.Error())
				return
			}
		}

		{
			// TODO 需要支持sudo
			var cmd = exec.Command(firewallCmd, "--add-port="+port)
			err = cmd.Run()
			if err != nil {
				remotelogs.Warn("FIREWALLD", "'"+cmd.String()+"': "+err.Error())
				return
			}
		}
	}
}
