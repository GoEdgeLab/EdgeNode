package nodes

import (
	"context"
	"errors"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/iwind/TeaGo/logs"
	"net"
	"sync"
)

type Listener struct {
	group       *serverconfigs.ServerGroup
	isListening bool
	listener    ListenerImpl // 监听器

	locker sync.RWMutex
}

func NewListener() *Listener {
	return &Listener{}
}

func (this *Listener) Reload(group *serverconfigs.ServerGroup) {
	this.locker.Lock()
	defer this.locker.Unlock()
	this.group = group
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

	netListener, err := this.createListener()
	if err != nil {
		return err
	}

	switch protocol {
	case serverconfigs.ProtocolHTTP, serverconfigs.ProtocolHTTP4, serverconfigs.ProtocolHTTP6:
		this.listener = &HTTPListener{
			Group:    this.group,
			Listener: netListener,
		}
	case serverconfigs.ProtocolHTTPS, serverconfigs.ProtocolHTTPS4, serverconfigs.ProtocolHTTPS6:
		this.listener = &HTTPListener{
			Group:    this.group,
			Listener: netListener,
		}
	case serverconfigs.ProtocolTCP, serverconfigs.ProtocolTCP4, serverconfigs.ProtocolTCP6:
		this.listener = &TCPListener{
			Group:    this.group,
			Listener: netListener,
		}
	case serverconfigs.ProtocolTLS, serverconfigs.ProtocolTLS4, serverconfigs.ProtocolTLS6:
		this.listener = &TCPListener{
			Group:    this.group,
			Listener: netListener,
		}
	case serverconfigs.ProtocolUnix:
		this.listener = &UnixListener{
			Group:    this.group,
			Listener: netListener,
		}
	case serverconfigs.ProtocolUDP:
		this.listener = &UDPListener{
			Group:    this.group,
			Listener: netListener,
		}
	default:
		return errors.New("unknown protocol '" + protocol + "'")
	}

	this.listener.Init()

	go func() {
		err := this.listener.Serve()
		if err != nil {
			logs.Println("[LISTENER]" + err.Error())
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

// 创建监听器
func (this *Listener) createListener() (net.Listener, error) {
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
