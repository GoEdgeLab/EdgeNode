package nodes

import (
	"github.com/TeaOSLab/EdgeNode/internal/configs"
	"github.com/iwind/TeaGo/lists"
	"github.com/iwind/TeaGo/logs"
	"sync"
)

var sharedListenerManager = NewListenerManager()

type ListenerManager struct {
	listenersMap map[string]*Listener // addr => *Listener
	locker       sync.Mutex
}

func NewListenerManager() *ListenerManager {
	return &ListenerManager{
		listenersMap: map[string]*Listener{},
	}
}

func (this *ListenerManager) Start(node *configs.NodeConfig) error {
	this.locker.Lock()
	defer this.locker.Unlock()

	// 所有的新地址
	groupAddrs := []string{}
	for _, group := range node.AvailableGroups() {
		addr := group.FullAddr()
		groupAddrs = append(groupAddrs, addr)
	}

	// 停掉老的
	for _, listener := range this.listenersMap {
		addr := listener.FullAddr()
		if !lists.ContainsString(groupAddrs, addr) {
			logs.Println("[LISTENER_MANAGER]close '" + addr + "'")
			_ = listener.Close()
		}
	}

	// 启动新的或修改老的
	for _, group := range node.AvailableGroups() {
		addr := group.FullAddr()
		listener, ok := this.listenersMap[addr]
		if ok {
			logs.Println("[LISTENER_MANAGER]reload '" + addr + "'")
			listener.Reload(group)
		} else {
			logs.Println("[LISTENER_MANAGER]listen '" + addr + "'")
			listener = NewListener()
			listener.Reload(group)
			err := listener.Listen()
			if err != nil {
				return err
			}
			this.listenersMap[addr] = listener
		}
	}

	return nil
}
