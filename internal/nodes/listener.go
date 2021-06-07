package nodes

import (
	"context"
	"errors"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"net"
	"sync"
)

type Listener struct {
	group       *serverconfigs.ServerGroup
	isListening bool
	listener    ListenerInterface // 监听器

	locker sync.RWMutex
}

func NewListener() *Listener {
	return &Listener{}
}

func (this *Listener) Reload(group *serverconfigs.ServerGroup) {
	this.locker.Lock()
	this.group = group
	if this.listener != nil {
		this.listener.Reload(group)
	}
	this.locker.Unlock()
}

func (this *Listener) FullAddr() string {
	if this.group != nil {
		return this.group.FullAddr()
	}
	return ""
}

func (this *Listener) Listen() error {
	if this.group == nil {
		return nil
	}
	protocol := this.group.Protocol()
	if protocol.IsUDPFamily() {
		return this.listenUDP()
	}
	return this.listenTCP()
}

func (this *Listener) listenTCP() error {
	if this.group == nil {
		return nil
	}
	protocol := this.group.Protocol()

	netListener, err := this.createTCPListener()
	if err != nil {
		return err
	}
	netListener = NewTrafficListener(netListener)
	events.On(events.EventQuit, func() {
		remotelogs.Println("LISTENER", "quit "+this.group.FullAddr())
		_ = netListener.Close()
	})

	switch protocol {
	case serverconfigs.ProtocolHTTP, serverconfigs.ProtocolHTTP4, serverconfigs.ProtocolHTTP6:
		this.listener = &HTTPListener{
			BaseListener: BaseListener{Group: this.group},
			Listener:     netListener,
		}
	case serverconfigs.ProtocolHTTPS, serverconfigs.ProtocolHTTPS4, serverconfigs.ProtocolHTTPS6:
		this.listener = &HTTPListener{
			BaseListener: BaseListener{Group: this.group},
			Listener:     netListener,
		}
	case serverconfigs.ProtocolTCP, serverconfigs.ProtocolTCP4, serverconfigs.ProtocolTCP6:
		this.listener = &TCPListener{
			BaseListener: BaseListener{Group: this.group},
			Listener:     netListener,
		}
	case serverconfigs.ProtocolTLS, serverconfigs.ProtocolTLS4, serverconfigs.ProtocolTLS6:
		this.listener = &TCPListener{
			BaseListener: BaseListener{Group: this.group},
			Listener:     netListener,
		}
	case serverconfigs.ProtocolUnix:
		this.listener = &UnixListener{
			BaseListener: BaseListener{Group: this.group},
			Listener:     netListener,
		}
	default:
		return errors.New("unknown protocol '" + protocol.String() + "'")
	}

	this.listener.Init()

	go func() {
		err := this.listener.Serve()
		if err != nil {
			// 在这里屏蔽accept错误，防止在优雅关闭的时候有多余的提示
			opErr, ok := err.(*net.OpError)
			if ok && opErr.Op == "accept" {
				return
			}

			// 打印其他错误
			remotelogs.Error("LISTENER", err.Error())
		}
	}()

	return nil
}

func (this *Listener) listenUDP() error {
	listener, err := this.createUDPListener()
	if err != nil {
		return err
	}
	events.On(events.EventQuit, func() {
		remotelogs.Println("LISTENER", "quit "+this.group.FullAddr())
		_ = listener.Close()
	})

	this.listener = &UDPListener{
		BaseListener: BaseListener{Group: this.group},
		Listener:     listener,
	}

	go func() {
		err := this.listener.Serve()
		if err != nil {
			remotelogs.Error("LISTENER", err.Error())
		}
	}()

	return nil
}

func (this *Listener) Close() error {
	if this.listener == nil {
		return nil
	}
	return this.listener.Close()
}

// 创建TCP监听器
func (this *Listener) createTCPListener() (net.Listener, error) {
	listenConfig := net.ListenConfig{
		Control:   nil,
		KeepAlive: 0,
	}

	switch this.group.Protocol() {
	case serverconfigs.ProtocolHTTP4, serverconfigs.ProtocolHTTPS4, serverconfigs.ProtocolTLS4:
		return listenConfig.Listen(context.Background(), "tcp4", this.group.Addr())
	case serverconfigs.ProtocolHTTP6, serverconfigs.ProtocolHTTPS6, serverconfigs.ProtocolTLS6:
		return listenConfig.Listen(context.Background(), "tcp6", this.group.Addr())
	}

	return listenConfig.Listen(context.Background(), "tcp", this.group.Addr())
}

// 创建UDP监听器
func (this *Listener) createUDPListener() (*net.UDPConn, error) {
	// TODO 将来支持udp4/udp6
	addr, err := net.ResolveUDPAddr("udp", this.group.Addr())
	if err != nil {
		return nil, err
	}
	return net.ListenUDP("udp", addr)
}
