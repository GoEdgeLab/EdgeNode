package nodes

import (
	"context"
	"errors"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
	"net"
	"strings"
	"sync"
)

type Listener struct {
	group    *serverconfigs.ServerAddressGroup
	listener ListenerInterface // 监听器

	locker sync.RWMutex
}

func NewListener() *Listener {
	return &Listener{}
}

func (this *Listener) Reload(group *serverconfigs.ServerAddressGroup) {
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
	var protocol = this.group.Protocol()
	if protocol.IsUDPFamily() {
		return this.listenUDP()
	}
	return this.listenTCP()
}

func (this *Listener) listenTCP() error {
	if this.group == nil {
		return nil
	}
	var protocol = this.group.Protocol()

	tcpListener, err := this.createTCPListener()
	if err != nil {
		return err
	}
	var netListener = NewClientListener(tcpListener, protocol.IsHTTPFamily() || protocol.IsHTTPSFamily())
	events.OnKey(events.EventQuit, this, func() {
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
		netListener.SetIsTLS(true)
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
		netListener.SetIsTLS(true)
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

	goman.New(func() {
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
	})

	return nil
}

func (this *Listener) listenUDP() error {
	var addr = this.group.Addr()

	var ipv4PacketListener *ipv4.PacketConn
	var ipv6PacketListener *ipv6.PacketConn

	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}

	if len(host) == 0 {
		// ipv4
		ipv4Listener, err := this.createUDPIPv4Listener()
		if err == nil {
			ipv4PacketListener = ipv4.NewPacketConn(ipv4Listener)
		} else {
			remotelogs.Error("LISTENER", "create udp ipv4 listener '"+addr+"': "+err.Error())
		}

		// ipv6
		ipv6Listener, err := this.createUDPIPv6Listener()
		if err == nil {
			ipv6PacketListener = ipv6.NewPacketConn(ipv6Listener)
		} else {
			remotelogs.Error("LISTENER", "create udp ipv6 listener '"+addr+"': "+err.Error())
		}
	} else if strings.Contains(host, ":") { // ipv6
		ipv6Listener, err := this.createUDPIPv6Listener()
		if err == nil {
			ipv6PacketListener = ipv6.NewPacketConn(ipv6Listener)
		} else {
			remotelogs.Error("LISTENER", "create udp ipv6 listener '"+addr+"': "+err.Error())
		}
	} else { // ipv4
		ipv4Listener, err := this.createUDPIPv4Listener()
		if err == nil {
			ipv4PacketListener = ipv4.NewPacketConn(ipv4Listener)
		} else {
			remotelogs.Error("LISTENER", "create udp ipv4 listener '"+addr+"': "+err.Error())
		}
	}

	events.OnKey(events.EventQuit, this, func() {
		remotelogs.Println("LISTENER", "quit "+this.group.FullAddr())

		if ipv4PacketListener != nil {
			_ = ipv4PacketListener.Close()
		}

		if ipv6PacketListener != nil {
			_ = ipv6PacketListener.Close()
		}
	})

	this.listener = &UDPListener{
		BaseListener: BaseListener{Group: this.group},
		IPv4Listener: ipv4PacketListener,
		IPv6Listener: ipv6PacketListener,
	}

	goman.New(func() {
		err := this.listener.Serve()
		if err != nil {
			remotelogs.Error("LISTENER", err.Error())
		}
	})

	return nil
}

func (this *Listener) Close() error {
	events.Remove(this)

	if this.listener == nil {
		return nil
	}
	return this.listener.Close()
}

// 创建TCP监听器
func (this *Listener) createTCPListener() (net.Listener, error) {
	var listenConfig = net.ListenConfig{
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

// 创建UDP IPv4监听器
func (this *Listener) createUDPIPv4Listener() (*net.UDPConn, error) {
	addr, err := net.ResolveUDPAddr("udp", this.group.Addr())
	if err != nil {
		return nil, err
	}
	return net.ListenUDP("udp4", addr)
}

// 创建UDP监听器
func (this *Listener) createUDPIPv6Listener() (*net.UDPConn, error) {
	addr, err := net.ResolveUDPAddr("udp", this.group.Addr())
	if err != nil {
		return nil, err
	}
	return net.ListenUDP("udp6", addr)
}
